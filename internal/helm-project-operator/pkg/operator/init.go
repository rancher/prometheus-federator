package operator

import (
	"context"
	"fmt"

	"github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/controllers"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/controllers/common"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"k8s.io/client-go/tools/clientcmd"

	commoncrds "github.com/rancher/prometheus-federator/internal/helmcommon/pkg/crds"
)

// Init sets up a new Helm Project Operator with the provided options and configuration
func Init(
	ctx context.Context,
	systemNamespace string,
	cfg clientcmd.ClientConfig,
	opts common.Options,
	crds []crd.CRD,
) error {
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

	if err := commoncrds.CreateFrom(ctx, clientConfig, crds); err != nil {
		return err
	}
	return controllers.Register(ctx, systemNamespace, cfg, opts)
}
