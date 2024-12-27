package operator

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/rancher/helm-locker/pkg/controllers"
	"github.com/rancher/helm-locker/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"k8s.io/client-go/tools/clientcmd"
)

type ControllerOptions struct {
	ClientConfig clientcmd.ClientConfig

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

func Run(ctx context.Context, options ControllerOptions) error {
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

	if err := crd.Create(ctx, clientConfig); err != nil {
		return err
	}

	if err := controllers.Register(
		ctx,
		options.Namespace,
		options.ControllerName,
		options.NodeName,
		options.ClientConfig,
	); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}
