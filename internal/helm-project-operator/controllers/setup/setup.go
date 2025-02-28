package setup

import (
	"context"
	"time"

	k3shelm "github.com/k3s-io/helm-controller/pkg/generated/controllers/helm.cattle.io"
	k3shelmcontroller "github.com/k3s-io/helm-controller/pkg/generated/controllers/helm.cattle.io/v1"
	"github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	helmlocker "github.com/rancher/prometheus-federator/internal/helm-locker/generated/controllers/helm.cattle.io"
	helmlockercontroller "github.com/rancher/prometheus-federator/internal/helm-locker/generated/controllers/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-locker/objectset"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	helmproject "github.com/rancher/prometheus-federator/internal/helm-project-operator/generated/controllers/helm.cattle.io"
	helmprojectcontroller "github.com/rancher/prometheus-federator/internal/helm-project-operator/generated/controllers/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/apply"
	batch "github.com/rancher/wrangler/v3/pkg/generated/controllers/batch"
	batchcontroller "github.com/rancher/wrangler/v3/pkg/generated/controllers/batch/v1"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	corecontroller "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/networking.k8s.io"
	networkingcontroller "github.com/rancher/wrangler/v3/pkg/generated/controllers/networking.k8s.io/v1"
	rbac "github.com/rancher/wrangler/v3/pkg/generated/controllers/rbac"
	rbaccontroller "github.com/rancher/wrangler/v3/pkg/generated/controllers/rbac/v1"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"github.com/rancher/wrangler/v3/pkg/start"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type AppContext struct {
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

func (a *AppContext) Start(ctx context.Context) error {
	return start.All(ctx, 50, a.starters...)
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

func NewAppContext(cfg clientcmd.ClientConfig, systemNamespace string, opts common.Options) (*AppContext, error) {
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

	return &AppContext{
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
