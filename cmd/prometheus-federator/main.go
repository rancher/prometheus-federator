package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/rancher/prometheus-federator/internal/hack"
	command "github.com/rancher/prometheus-federator/internal/helm-project-operator/cli"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/operator"
	"github.com/rancher/prometheus-federator/pkg/version"
	"github.com/rancher/wrangler/v3/pkg/crd"
	_ "github.com/rancher/wrangler/v3/pkg/generated/controllers/apiextensions.k8s.io"
	_ "github.com/rancher/wrangler/v3/pkg/generated/controllers/networking.k8s.io"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
)

const (
	// HelmAPIVersion is the spec.helmApiVersion corresponding to project monitoring releases managed by this operator.
	HelmAPIVersion = "monitoring.cattle.io/v1alpha1"

	// ReleaseName is the release name this operator uses to prefix managed project monitoring releases.
	ReleaseName = "monitoring"
)

var (
	// SystemNamespaces is the system namespaces scoped for managed project monitoring releases.
	SystemNamespaces = []string{"kube-system", "cattle-monitoring-system", "cattle-dashboards"}

	debugConfig command.DebugConfig
)

type PrometheusFederator struct {
	// Note: all Project Operator are expected to provide these RuntimeOptions
	common.RuntimeOptions

	Kubeconfig string `usage:"Kubeconfig file"`
}

func (f *PrometheusFederator) Run(cmd *cobra.Command, _ []string) error {
	go func() {
		// required to set up healthz and pprof handlers
		log.Println(http.ListenAndServe("localhost:80", nil))
	}()
	debugConfig.MustSetupDebug()

	cfg := kubeconfig.GetNonInteractiveClientConfig(f.Kubeconfig)

	ctx := cmd.Context()

	// Note : for SURE-8872 we introduced some new flags, we keep the logic to handle them here,
	// but end-users experiencing the same issues will be unaffected without using these values

	update := os.Getenv("MANAGE_CRD_UPDATES") == "true"
	clientConfig, err := cfg.ClientConfig()
	if err != nil {
		logrus.Fatalf("Failed to get client config from runtime : %s", err)
	}

	// by default only include crds required by the runtime
	managedCrds := common.ManagedCRDsFromRuntime(f.RuntimeOptions)

	// if users turn off crd management, no new crds will be applied if a matching CRD with the same GVK
	// and version already exists
	if !update {
		managedCrds, err = filter(clientConfig, managedCrds)
		if err != nil {
			return err
		}
	}

	if err := operator.Init(ctx, f.Namespace, cfg, common.Options{
		OperatorOptions: common.OperatorOptions{
			HelmAPIVersion:   HelmAPIVersion,
			ReleaseName:      ReleaseName,
			SystemNamespaces: SystemNamespaces,
			Singleton:        true, // indicates only one HelmChart can be registered per project defined
		},
		RuntimeOptions: f.RuntimeOptions,
	},
		managedCrds,
	); err != nil {
		return err
	}

	<-cmd.Context().Done()
	return nil
}

func filter(clientConfig *rest.Config, crds []crd.CRD) ([]crd.CRD, error) {
	factory, err := crd.NewFactoryFromClient(clientConfig)
	if err != nil {
		return nil, err
	}
	crdClientSet := factory.CRDClient.(*clientset.Clientset)
	client := crdClientSet.ApiextensionsV1().CustomResourceDefinitions()
	filteredCrd, err := hack.FilterMissingCRDs(client, crds)
	if err != nil {
		return nil, err
	}
	return filteredCrd, nil
}

func main() {
	cmd := command.Command(&PrometheusFederator{}, cobra.Command{
		Version: version.FriendlyVersion(),
	})
	cmd = command.AddDebug(cmd, &debugConfig)
	command.Main(cmd)
}
