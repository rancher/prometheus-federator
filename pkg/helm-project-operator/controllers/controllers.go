package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/hardened"
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/project"
	helmproject "github.com/rancher/prometheus-federator/pkg/helm-project-operator/generated/controllers/helm.cattle.io"
	helmprojectcontroller "github.com/rancher/prometheus-federator/pkg/helm-project-operator/generated/controllers/helm.cattle.io/v1alpha1"

	"github.com/k3s-io/helm-controller/pkg/controllers/chart"
	k3shelm "github.com/k3s-io/helm-controller/pkg/generated/controllers/helm.cattle.io"
	k3shelmcontroller "github.com/k3s-io/helm-controller/pkg/generated/controllers/helm.cattle.io/v1"
	"github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/prometheus-federator/pkg/helm-locker/controllers/release"
	helmlocker "github.com/rancher/prometheus-federator/pkg/helm-locker/generated/controllers/helm.cattle.io"
	helmlockercontroller "github.com/rancher/prometheus-federator/pkg/helm-locker/generated/controllers/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/pkg/helm-locker/objectset"
	"github.com/rancher/wrangler/pkg/apply"
	batch "github.com/rancher/wrangler/pkg/generated/controllers/batch"
	batchcontroller "github.com/rancher/wrangler/pkg/generated/controllers/batch/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	corecontroller "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/generated/controllers/networking.k8s.io"
	networkingcontroller "github.com/rancher/wrangler/pkg/generated/controllers/networking.k8s.io/v1"
	rbac "github.com/rancher/wrangler/pkg/generated/controllers/rbac"
	rbaccontroller "github.com/rancher/wrangler/pkg/generated/controllers/rbac/v1"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/leader"
	"github.com/rancher/wrangler/pkg/ratelimit"
	"github.com/rancher/wrangler/pkg/schemes"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type appContext struct {
	helmprojectcontroller.Interface

	Dynamic    dynamic.Interface
	K8s        kubernetes.Interface
	Core       corecontroller.Interface
	Networking networkingcontroller.Interface

	HelmLocker        helmlockercontroller.Interface
	ObjectSetRegister objectset.LockableRegister
	ObjectSetHandler  *controller.SharedHandler

	HelmController k3shelmcontroller.Interface
	Batch          batchcontroller.Interface
	RBAC           rbaccontroller.Interface

	Apply            apply.Apply
	EventBroadcaster record.EventBroadcaster

	ClientConfig clientcmd.ClientConfig
	starters     []start.Starter
}

func (a *appContext) start(ctx context.Context) error {
	return start.All(ctx, 50, a.starters...)
}

// Register registers all controllers for the Helm Project Operator based on the provided options
func Register(ctx context.Context, systemNamespace string, cfg clientcmd.ClientConfig, opts common.Options) error {
	if len(systemNamespace) == 0 {
		return errors.New("cannot start controllers on system namespace: system namespace not provided")
	}
	// always add the systemNamespace to the systemNamespaces provided
	opts.SystemNamespaces = append(opts.SystemNamespaces, systemNamespace)

	// parse values.yaml and questions.yaml from file
	valuesYaml, questionsYaml, err := parseValuesAndQuestions(opts.ChartContent)
	if err != nil {
		logrus.Fatal(err)
	}

	appCtx, err := newContext(cfg, systemNamespace, opts)
	if err != nil {
		return err
	}

	appCtx.EventBroadcaster.StartLogging(logrus.Debugf)
	appCtx.EventBroadcaster.StartRecordingToSink(&typedv1.EventSinkImpl{
		Interface: appCtx.K8s.CoreV1().Events(systemNamespace),
	})
	recorder := appCtx.EventBroadcaster.NewRecorder(schemes.All, corev1.EventSource{
		Component: "helm-project-operator",
		Host:      opts.NodeName,
	})

	if !opts.DisableHardening {
		hardeningOpts, err := common.LoadHardeningOptionsFromFile(opts.HardeningOptionsFile)
		if err != nil {
			return err
		}
		hardened.Register(ctx,
			appCtx.Apply,
			hardeningOpts,
			// watches
			appCtx.Core.Namespace(),
			appCtx.Core.Namespace().Cache(),
			// generates
			appCtx.Core.ServiceAccount(),
			appCtx.Networking.NetworkPolicy(),
		)
	}

	projectGetter := namespace.Register(ctx,
		appCtx.Apply,
		systemNamespace,
		valuesYaml,
		questionsYaml,
		opts,
		// watches and generates
		appCtx.Core.Namespace(),
		appCtx.Core.Namespace().Cache(),
		appCtx.Core.ConfigMap(),
		// enqueues
		appCtx.ProjectHelmChart(),
		appCtx.ProjectHelmChart().Cache(),
		appCtx.Dynamic,
	)

	if len(opts.ControllerName) == 0 {
		opts.ControllerName = "helm-project-operator"
	}

	valuesOverride, err := common.LoadValuesOverrideFromFile(opts.ValuesOverrideFile)
	if err != nil {
		return err
	}
	project.Register(ctx,
		systemNamespace,
		opts,
		valuesOverride,
		appCtx.Apply,
		// watches
		appCtx.ProjectHelmChart(),
		appCtx.ProjectHelmChart().Cache(),
		appCtx.Core.ConfigMap(),
		appCtx.Core.ConfigMap().Cache(),
		appCtx.RBAC.Role(),
		appCtx.RBAC.Role().Cache(),
		appCtx.RBAC.ClusterRoleBinding(),
		appCtx.RBAC.ClusterRoleBinding().Cache(),
		// watches and generates
		appCtx.HelmController.HelmChart(),
		appCtx.HelmLocker.HelmRelease(),
		appCtx.Core.Namespace(),
		appCtx.Core.Namespace().Cache(),
		appCtx.RBAC.RoleBinding(),
		appCtx.RBAC.RoleBinding().Cache(),
		projectGetter,
	)

	if !opts.DisableEmbeddedHelmLocker {
		logrus.Infof("Registering embedded Helm Locker...")
		release.Register(ctx,
			systemNamespace,
			opts.ControllerName,
			appCtx.HelmLocker.HelmRelease(),
			appCtx.HelmLocker.HelmRelease().Cache(),
			appCtx.Core.Secret(),
			appCtx.Core.Secret().Cache(),
			appCtx.K8s,
			appCtx.ObjectSetRegister,
			appCtx.ObjectSetHandler,
			recorder,
		)
	}

	if !opts.DisableEmbeddedHelmController {
		logrus.Infof("Registering embedded Helm Controller...")
		chart.Register(ctx,
			systemNamespace,
			opts.ControllerName,
			appCtx.K8s,
			appCtx.Apply,
			recorder,
			appCtx.HelmController.HelmChart(),
			appCtx.HelmController.HelmChart().Cache(),
			appCtx.HelmController.HelmChartConfig(),
			appCtx.HelmController.HelmChartConfig().Cache(),
			appCtx.Batch.Job(),
			appCtx.Batch.Job().Cache(),
			appCtx.RBAC.ClusterRoleBinding(),
			appCtx.Core.ServiceAccount(),
			appCtx.Core.ConfigMap())
	}

	leader.RunOrDie(ctx, systemNamespace, fmt.Sprintf("helm-project-operator-%s-lock", opts.ReleaseName), appCtx.K8s, func(ctx context.Context) {
		if err := appCtx.start(ctx); err != nil {
			logrus.Fatal(err)
		}
		logrus.Info("All controllers have been started")
	})

	return nil
}

func controllerFactory(rest *rest.Config) (controller.SharedControllerFactory, error) {
	rateLimit := workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 60*time.Second)
	clientFactory, err := client.NewSharedClientFactory(rest, nil)
	if err != nil {
		return nil, err
	}

	cacheFactory := cache.NewSharedCachedFactory(clientFactory, nil)
	return controller.NewSharedControllerFactory(cacheFactory, &controller.SharedControllerFactoryOptions{
		DefaultRateLimiter: rateLimit,
		DefaultWorkers:     50,
	}), nil
}

func newContext(cfg clientcmd.ClientConfig, systemNamespace string, opts common.Options) (*appContext, error) {
	client, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}
	client.RateLimiter = ratelimit.None

	dynamic, err := dynamic.NewForConfig(client)
	if err != nil {
		return nil, err
	}

	k8s, err := kubernetes.NewForConfig(client)
	if err != nil {
		return nil, err
	}

	discovery, err := discovery.NewDiscoveryClientForConfig(client)
	if err != nil {
		return nil, err
	}

	apply := apply.New(discovery, apply.NewClientFactory(client))

	scf, err := controllerFactory(client)
	if err != nil {
		return nil, err
	}

	// Shared Controllers

	core, err := core.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		SharedControllerFactory: scf,
	})
	if err != nil {
		return nil, err
	}
	corev := core.Core().V1()

	networking, err := networking.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		SharedControllerFactory: scf,
	})
	if err != nil {
		return nil, err
	}
	networkingv := networking.Networking().V1()

	// Helm Project Controller

	var namespace string // by default, this is unset so we watch everything
	if len(opts.ProjectLabel) == 0 {
		// we only need to watch the systemNamespace
		namespace = systemNamespace
	}

	helmproject, err := helmproject.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		SharedControllerFactory: scf,
		Namespace:               namespace,
	})
	if err != nil {
		return nil, err
	}
	helmprojectv := helmproject.Helm().V1alpha1()

	// Helm Locker Controllers - should be scoped to the system namespace only

	objectSet, objectSetRegister, objectSetHandler := objectset.NewLockableRegister("object-set-register", apply, scf, discovery, nil)

	helmlocker, err := helmlocker.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		SharedControllerFactory: scf,
		Namespace:               systemNamespace,
	})
	if err != nil {
		return nil, err
	}
	helmlockerv := helmlocker.Helm().V1alpha1()

	// Helm Controllers - should be scoped to the system namespace only

	helm, err := k3shelm.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		SharedControllerFactory: scf,
		Namespace:               systemNamespace,
	})
	if err != nil {
		return nil, err
	}
	helmv := helm.Helm().V1()

	batch, err := batch.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		SharedControllerFactory: scf,
		Namespace:               systemNamespace,
	})
	if err != nil {
		return nil, err
	}
	batchv := batch.Batch().V1()

	rbac, err := rbac.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		SharedControllerFactory: scf,
		Namespace:               systemNamespace,
	})
	if err != nil {
		return nil, err
	}
	rbacv := rbac.Rbac().V1()

	return &appContext{
		Interface: helmprojectv,

		Dynamic:    dynamic,
		K8s:        k8s,
		Core:       corev,
		Networking: networkingv,

		HelmLocker:        helmlockerv,
		ObjectSetRegister: objectSetRegister,
		ObjectSetHandler:  objectSetHandler,

		HelmController: helmv,
		Batch:          batchv,
		RBAC:           rbacv,

		Apply:            apply.WithSetOwnerReference(false, false),
		EventBroadcaster: record.NewBroadcaster(),

		ClientConfig: cfg,
		starters: []start.Starter{
			core,
			networking,
			batch,
			rbac,
			helm,
			objectSet,
			helmlocker,
			helmproject,
		},
	}, nil
}
