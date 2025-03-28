package main

import (
	_ "embed"
	"log"
	"net/http"
	_ "net/http/pprof"

	command "github.com/rancher/prometheus-federator/internal/helm-project-operator/cli"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/operator"
	"github.com/rancher/prometheus-federator/pkg/version"
	_ "github.com/rancher/wrangler/v3/pkg/generated/controllers/apiextensions.k8s.io"
	_ "github.com/rancher/wrangler/v3/pkg/generated/controllers/networking.k8s.io"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/spf13/cobra"
)

const (
	// DummyHelmAPIVersion is the spec.helmApiVersion corresponding to the dummy example-chart
	DummyHelmAPIVersion = "dummy.cattle.io/v1alpha1"

	// DummyReleaseName is the release name corresponding to the operator that deploys the dummy example-chart
	DummyReleaseName = "dummy"
)

var (
	// DummySystemNamespaces is the system namespaces scoped for the dummy example-chart.
	DummySystemNamespaces = []string{"kube-system"}

	debugConfig command.DebugConfig
	//go:embed fs/example-chart.tgz.base64
	dummyChart string
)

type DummyOperator struct {
	// Note: all Project Operator are expected to provide these RuntimeOptions
	common.RuntimeOptions

	Kubeconfig string `usage:"Kubeconfig file"`
}

func (o *DummyOperator) Run(cmd *cobra.Command, _ []string) error {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	debugConfig.MustSetupDebug()

	cfg := kubeconfig.GetNonInteractiveClientConfig(o.Kubeconfig)

	ctx := cmd.Context()

	managedCrds := common.ManagedCRDsFromRuntime(o.RuntimeOptions)

	if err := operator.Init(ctx, o.Namespace, cfg, common.Options{
		OperatorOptions: common.OperatorOptions{
			HelmAPIVersion:   DummyHelmAPIVersion,
			ReleaseName:      DummyReleaseName,
			SystemNamespaces: DummySystemNamespaces,
			ChartContent:     dummyChart,
			Singleton:        false,
		},
		RuntimeOptions: o.RuntimeOptions,
	},
		managedCrds,
	); err != nil {
		return err
	}

	<-cmd.Context().Done()
	return nil
}

func main() {
	cmd := command.Command(&DummyOperator{}, cobra.Command{
		Version: version.FriendlyVersion(),
	})
	cmd = command.AddDebug(cmd, &debugConfig)
	command.Main(cmd)
}
