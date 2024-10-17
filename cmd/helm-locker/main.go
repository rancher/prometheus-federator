//go:build helm_locker

package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/rancher/prometheus-federator/pkg/helm-locker/controllers"
	"github.com/rancher/prometheus-federator/pkg/helm-locker/crd"
	"github.com/rancher/prometheus-federator/pkg/version"
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

type HelmLocker struct {
	Kubeconfig     string `usage:"Kubeconfig file" env:"KUBECONFIG"`
	Namespace      string `usage:"Namespace to watch for HelmReleases" default:"cattle-helm-system" env:"NAMESPACE"`
	ControllerName string `usage:"Unique name to identify this controller that is added to all HelmReleases tracked by this controller" default:"helm-locker" env:"CONTROLLER_NAME"`
	NodeName       string `usage:"Name of the node this controller is running on" env:"NODE_NAME"`
}

func (a *HelmLocker) Run(cmd *cobra.Command, _ []string) error {
	if len(a.Namespace) == 0 {
		return fmt.Errorf("helm-locker can only be started in a single namespace")
	}

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	debugConfig.MustSetupDebug()

	cfg := kubeconfig.GetNonInteractiveClientConfig(a.Kubeconfig)
	clientConfig, err := cfg.ClientConfig()
	if err != nil {
		return err
	}
	clientConfig.RateLimiter = ratelimit.None

	ctx := cmd.Context()
	if err := crd.Create(ctx, clientConfig); err != nil {
		return err
	}

	if err := controllers.Register(ctx, a.Namespace, a.ControllerName, a.NodeName, cfg); err != nil {
		return err
	}

	<-cmd.Context().Done()
	return nil
}

func main() {
	cmd := command.Command(&HelmLocker{}, cobra.Command{
		Version: version.FriendlyVersion(),
	})
	cmd = command.AddDebug(cmd, &debugConfig)
	command.Main(cmd)
}
