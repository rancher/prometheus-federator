package controllers

import (
	"context"
	"errors"
	"time"

	"github.com/rancher/helm-locker/pkg/controllers/release"
	"github.com/rancher/helm-locker/pkg/generated/controllers/helm.cattle.io"
	helmcontroller "github.com/rancher/helm-locker/pkg/generated/controllers/helm.cattle.io/v1alpha1"
	"github.com/rancher/helm-locker/pkg/objectset"
	"github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/apply"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	corecontroller "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/leader"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"github.com/rancher/wrangler/v3/pkg/schemes"
	"github.com/rancher/wrangler/v3/pkg/start"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type appContext struct {
	helmcontroller.Interface

	K8s  kubernetes.Interface
	Core corecontroller.Interface

	Apply apply.Apply

	ObjectSetRegister objectset.LockableRegister
	ObjectSetHandler  *controller.SharedHandler

	EventBroadcaster record.EventBroadcaster

	starters []start.Starter
}

func (a *appContext) start(ctx context.Context) error {
	return start.All(ctx, 50, a.starters...)
}

func Register(ctx context.Context, systemNamespace, controllerName, nodeName string, cfg clientcmd.ClientConfig) error {
	if len(systemNamespace) == 0 {
		return errors.New("cannot start controllers on system namespace: system namespace not provided")
	}

	appCtx, err := newContext(ctx, systemNamespace, cfg)
	if err != nil {
		return err
	}

	appCtx.EventBroadcaster.StartLogging(logrus.Debugf)
	appCtx.EventBroadcaster.StartRecordingToSink(&typedv1.EventSinkImpl{
		Interface: appCtx.K8s.CoreV1().Events(systemNamespace),
	})
	recorder := appCtx.EventBroadcaster.NewRecorder(schemes.All, corev1.EventSource{
		Component: "helm-locker",
		Host:      nodeName,
	})

	if len(controllerName) == 0 {
		controllerName = "helm-locker"
	}

	// TODO: Register all controllers
	release.Register(ctx,
		systemNamespace,
		controllerName,
		appCtx.HelmRelease(),
		appCtx.HelmRelease().Cache(),
		appCtx.Core.Secret(),
		appCtx.Core.Secret().Cache(),
		appCtx.K8s,
		appCtx.ObjectSetRegister,
		appCtx.ObjectSetHandler,
		recorder,
	)

	leader.RunOrDie(ctx, systemNamespace, "helm-locker-lock", appCtx.K8s, func(ctx context.Context) {
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

func newContext(_ context.Context, systemNamespace string, cfg clientcmd.ClientConfig) (*appContext, error) {
	client, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}
	client.RateLimiter = ratelimit.None

	k8s, err := kubernetes.NewForConfig(client)
	if err != nil {
		return nil, err
	}

	discovery, err := discovery.NewDiscoveryClientForConfig(client)
	if err != nil {
		return nil, err
	}

	scf, err := controllerFactory(client)
	if err != nil {
		return nil, err
	}

	core, err := core.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		SharedControllerFactory: scf,
	})
	if err != nil {
		return nil, err
	}
	corev := core.Core().V1()

	helm, err := helm.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		Namespace:               systemNamespace,
		SharedControllerFactory: scf,
	})
	if err != nil {
		return nil, err
	}
	helmv := helm.Helm().V1alpha1()

	apply := apply.New(discovery, apply.NewClientFactory(client))

	objectSet, objectSetRegister, objectSetHandler := objectset.NewLockableRegister("object-set-register", apply, scf, discovery, nil)

	return &appContext{
		Interface: helmv,

		K8s:  k8s,
		Core: corev,

		Apply: apply,

		ObjectSetRegister: objectSetRegister,
		ObjectSetHandler:  objectSetHandler,

		EventBroadcaster: record.NewBroadcaster(),

		starters: []start.Starter{
			objectSet,
			core,
			helm,
		},
	}, nil
}
