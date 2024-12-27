package operator

import (
	"context"
	"fmt"

	"github.com/rancher/helm-project-operator/pkg/controllers"
	"github.com/rancher/helm-project-operator/pkg/controllers/common"
	"github.com/rancher/helm-project-operator/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"k8s.io/client-go/tools/clientcmd"
)

// Init sets up a new Helm Project Operator with the provided options and configuration
func Init(ctx context.Context, systemNamespace string, cfg clientcmd.ClientConfig, opts common.Options) error {
	if systemNamespace == "" {
		return fmt.Errorf("system namespace was not specified, unclear where to place HelmCharts or HelmReleases")
	}
	if err := opts.Validate(); err != nil {
		return err
	}

	clientConfig, err := cfg.ClientConfig()
	if err != nil {
		return err
	}
	clientConfig.RateLimiter = ratelimit.None

	if err := crd.Create(ctx, clientConfig); err != nil {
		return err
	}

	return controllers.Register(ctx, systemNamespace, cfg, opts)
}
