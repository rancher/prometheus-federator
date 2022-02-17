package controllers

import (
	"context"
	"time"

	"github.com/aiyengar2/prometheus-federator/pkg/controllers/project"
	"github.com/aiyengar2/prometheus-federator/pkg/generated/controllers/monitoring.cattle.io"
	monitoringcontrollers "github.com/aiyengar2/prometheus-federator/pkg/generated/controllers/monitoring.cattle.io/v1alpha1"
	prometheusoperator "github.com/aiyengar2/prometheus-federator/pkg/generated/controllers/monitoring.coreos.com"
	prometheusoperatorcontrollers "github.com/aiyengar2/prometheus-federator/pkg/generated/controllers/monitoring.coreos.com/v1"
	"github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/generated/controllers/apps"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	corecontrollers "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/leader"
	"github.com/rancher/wrangler/pkg/ratelimit"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

type appContext struct {
	monitoringcontrollers.Interface

	K8s                kubernetes.Interface
	Core               corecontrollers.Interface
	PrometheusOperator prometheusoperatorcontrollers.Interface
	Apply              apply.Apply
	ClientConfig       clientcmd.ClientConfig
	starters           []start.Starter
}

func (a *appContext) start(ctx context.Context) error {
	return start.All(ctx, 50, a.starters...)
}

func Register(ctx context.Context, systemNamespace, clusterPrometheusName, clusterPrometheusNamespace string, cfg clientcmd.ClientConfig) error {
	appCtx, err := newContext(cfg)
	if err != nil {
		return err
	}

	if err := addData(systemNamespace, clusterPrometheusNamespace, appCtx); err != nil {
		return err
	}

	// TODO: Register all controllers
	project.Register(ctx,
		systemNamespace,
		clusterPrometheusName,
		clusterPrometheusNamespace,
		appCtx.Apply,
		appCtx.Project(),
		appCtx.Project().Cache(),
		appCtx.PrometheusOperator.Prometheus(),
		appCtx.PrometheusOperator.Alertmanager(),
		appCtx.PrometheusOperator.PrometheusRule(),
		appCtx.PrometheusOperator.PodMonitor(),
		appCtx.Core.Namespace(),
		appCtx.Core.Namespace().Cache(),
	)

	leader.RunOrDie(ctx, systemNamespace, "prometheus-federator-lock", appCtx.K8s, func(ctx context.Context) {
		if err := appCtx.start(ctx); err != nil {
			logrus.Fatal(err)
		}
		logrus.Info("All controllers have been started")
	})

	return nil
}

func controllerFactory(rest *rest.Config) (controller.SharedControllerFactory, error) {
	rateLimit := workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 60*time.Second)
	workqueue.DefaultControllerRateLimiter()
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

func newContext(cfg clientcmd.ClientConfig) (*appContext, error) {
	client, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}
	client.RateLimiter = ratelimit.None

	apply, err := apply.NewForConfig(client)
	if err != nil {
		return nil, err
	}
	apply = apply.WithSetOwnerReference(false, false)

	k8s, err := kubernetes.NewForConfig(client)
	if err != nil {
		return nil, err
	}

	scf, err := controllerFactory(client)
	if err != nil {
		return nil, err
	}

	core, err := core.NewFactoryFromConfigWithOptions(client, &core.FactoryOptions{
		SharedControllerFactory: scf,
	})
	if err != nil {
		return nil, err
	}
	corev := core.Core().V1()

	monitoring, err := monitoring.NewFactoryFromConfigWithOptions(client, &apps.FactoryOptions{
		SharedControllerFactory: scf,
	})
	if err != nil {
		return nil, err
	}
	monitoringv := monitoring.Monitoring().V1alpha1()

	prometheusoperator, err := prometheusoperator.NewFactoryFromConfigWithOptions(client, &apps.FactoryOptions{
		SharedControllerFactory: scf,
	})
	if err != nil {
		return nil, err
	}
	prometheusoperatorv := prometheusoperator.Monitoring().V1()

	return &appContext{
		Interface: monitoringv,

		K8s:                k8s,
		Core:               corev,
		PrometheusOperator: prometheusoperatorv,
		Apply:              apply,
		ClientConfig:       cfg,
		starters: []start.Starter{
			core,
			prometheusoperator,
			monitoring,
		},
	}, nil
}
