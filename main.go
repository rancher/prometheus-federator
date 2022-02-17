package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/aiyengar2/prometheus-federator/pkg/controllers"
	"github.com/aiyengar2/prometheus-federator/pkg/crd"
	"github.com/aiyengar2/prometheus-federator/pkg/version"
	command "github.com/rancher/wrangler-cli"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/apiextensions.k8s.io"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/networking.k8s.io"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/ratelimit"
	"github.com/spf13/cobra"
)

var (
	debugConfig command.DebugConfig
)

type PrometheusFederator struct {
	Kubeconfig                 string `usage:"Kubeconfig file"`
	Namespace                  string `usage:"Namespace to watch for Projects" default:"cattle-project-monitoring-system" env:"NAMESPACE"`
	ClusterPrometheusName      string `usage:"Name of Cluster Prometheus" default:"rancher-monitoring-prometheus" env:"CLUSTER_PROMETHEUS_NAME"`
	ClusterPrometheusNamespace string `usage:"Namespace containing Cluster Prometheus" default:"cattle-monitoring-system" env:"CLUSTER_PROMETHEUS_NAMESPACE"`
}

func (p *PrometheusFederator) Run(cmd *cobra.Command, args []string) error {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	debugConfig.MustSetupDebug()

	cfg := kubeconfig.GetNonInteractiveClientConfig(p.Kubeconfig)
	clientConfig, err := cfg.ClientConfig()
	if err != nil {
		return err
	}
	clientConfig.RateLimiter = ratelimit.None

	ctx := cmd.Context()
	if err := crd.Create(ctx, clientConfig); err != nil {
		return err
	}

	if err := controllers.Register(ctx, p.Namespace, p.ClusterPrometheusName, p.ClusterPrometheusNamespace, cfg); err != nil {
		return err
	}

	<-cmd.Context().Done()
	return nil
}

func main() {
	cmd := command.Command(&PrometheusFederator{}, cobra.Command{
		Version: version.FriendlyVersion(),
	})
	cmd = command.AddDebug(cmd, &debugConfig)
	command.Main(cmd)
}
