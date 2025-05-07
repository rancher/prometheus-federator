package operator

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/rancher/prometheus-federator/internal/helm-locker/controllers"
	"github.com/sirupsen/logrus"

	// "github.com/rancher/prometheus-federator/internal/helm-locker/crd"
	commoncrds "github.com/rancher/prometheus-federator/internal/helmcommon/pkg/crds"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/leader"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"k8s.io/client-go/tools/clientcmd"
)

type ControllerOptions struct {
	ClientConfig   clientcmd.ClientConfig
	Namespace      string
	ControllerName string
	NodeName       string
	PprofEnabled   bool
}

func (c ControllerOptions) Validate() error {
	if len(c.Namespace) == 0 {
		return fmt.Errorf("helm-locker can only be started in a single namespace")
	}

	return nil
}

func Init(
	ctx context.Context,
	crds []crd.CRD,
	options ControllerOptions,
) error {
	if err := options.Validate(); err != nil {
		return err
	}

	if options.PprofEnabled {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	clientConfig, err := options.ClientConfig.ClientConfig()
	if err != nil {
		return err
	}

	clientConfig.RateLimiter = ratelimit.None

	if err := commoncrds.CreateFrom(ctx, clientConfig, crds); err != nil {
		return err
	}
	appCtx, err := controllers.NewContext(ctx, options.Namespace, options.ClientConfig)
	if err != nil {
		return err
	}

	if err := controllers.Register(
		ctx,
		appCtx,
		options.Namespace,
		options.ControllerName,
		options.NodeName,
		options.ClientConfig,
	); err != nil {
		return err
	}

	leader.RunOrDie(ctx, options.Namespace, "helm-locker-lock", appCtx.K8s, func(ctx context.Context) {
		if err := appCtx.Start(ctx); err != nil {
			logrus.Fatal(err)
		}
		logrus.Info("All controllers have been started")
	})

	<-ctx.Done()
	return nil
}
