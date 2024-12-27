package main

import (
	"context"
	_ "net/http/pprof"

	"github.com/rancher/helm-locker/pkg/operator"
	_ "github.com/rancher/wrangler/v3/pkg/generated/controllers/apiextensions.k8s.io"
	_ "github.com/rancher/wrangler/v3/pkg/generated/controllers/networking.k8s.io"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func BuildHelmLockerCommand() *cobra.Command {
	var kubeconfigVar string
	var namespace string
	var controllerName string
	var nodeName string
	var pprofEnabled bool
	viper.AutomaticEnv()
	cmd := &cobra.Command{
		Use: "helm-locker",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := kubeconfig.GetNonInteractiveClientConfig(kubeconfigVar)
			options := operator.ControllerOptions{
				Namespace:      namespace,
				ControllerName: controllerName,
				NodeName:       nodeName,
				ClientConfig:   cfg,
				PprofEnabled:   pprofEnabled,
			}
			if err := operator.Run(cmd.Context(), options); err != nil {
				return err
			}
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&kubeconfigVar, "kubeconfig", "k", "", "Kubeconfig file")
	flags.StringVar(&namespace, "namespace", "cattle-helm-system", "Namespace to watch for HelmReleases")
	flags.StringVar(&controllerName, "controller-name", "helm-locker", "Unique name to identify this controller that is added to all HelmReleases tracked by this controller")
	flags.StringVar(&nodeName, "node-name", "", "Name of the node this controller is running on")
	flags.BoolVarP(&pprofEnabled, "pprof", "p", false, "flag to enable pprof on port 6060")

	viper.BindPFlag("kubeconfig", flags.Lookup("KUBECONFIG"))
	viper.BindPFlag("namespace", flags.Lookup("NAMESPACE"))
	viper.BindPFlag("controller-name", flags.Lookup("CONTROLLER_NAME"))
	viper.BindPFlag("node-name", flags.Lookup("NODE_NAME"))
	return cmd
}

func main() {
	cmd := BuildHelmLockerCommand()
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		logrus.Errorf("failed to run helm locker : %s", err)
	}
}
