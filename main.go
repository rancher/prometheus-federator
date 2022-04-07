package main

import (
	_ "embed"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/aiyengar2/helm-project-operator/pkg/controllers/common"
	"github.com/aiyengar2/helm-project-operator/pkg/operator"
	"github.com/aiyengar2/prometheus-federator/pkg/version"
	command "github.com/rancher/wrangler-cli"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/apiextensions.k8s.io"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/networking.k8s.io"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/spf13/cobra"
)

const (
	HelmApiVersion = "monitoring.cattle.io/v1alpha1"
	ReleaseName    = "monitoring"
)

var (
	SystemNamespaces = []string{"kube-system", "cattle-monitoring-system", "cattle-dashboards"}

	//go:embed bin/rancher-project-monitoring/rancher-project-monitoring.tgz.base64
	base64TgzChart string

	debugConfig command.DebugConfig
)

type ProjectMonitoringValues struct {
}

type PrometheusFederator struct {
	// Note: all Project Operator are expected to provide these RuntimeOptions
	common.RuntimeOptions

	Kubeconfig string `usage:"Kubeconfig file"`
}

func (f *PrometheusFederator) Run(cmd *cobra.Command, args []string) error {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	debugConfig.MustSetupDebug()

	cfg := kubeconfig.GetNonInteractiveClientConfig(f.Kubeconfig)

	ctx := cmd.Context()

	if err := operator.Init(ctx, f.Namespace, cfg, common.Options{
		// These fields are provided by the Project Operator
		HelmApiVersion:   HelmApiVersion,
		ReleaseName:      ReleaseName,
		SystemNamespaces: SystemNamespaces,
		ChartContent:     base64TgzChart,

		// These fields are provided on runtime for all project operators
		ProjectLabel:            f.ProjectLabel,
		SystemProjectLabelValue: f.SystemProjectLabelValue,
		SystemDefaultRegistry:   f.SystemDefaultRegistry,
		CattleURL:               f.CattleURL,
		ClusterID:               f.ClusterID,
		NodeName:                f.NodeName,
	}); err != nil {
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
