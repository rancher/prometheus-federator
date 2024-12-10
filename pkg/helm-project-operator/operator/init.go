package operator

import (
	"context"
	"errors"
	"fmt"

	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers"
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/crd"
	"github.com/rancher/wrangler/pkg/clients"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/rancher/wrangler/pkg/ratelimit"
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

	k8sRuntimeType, err := identifyKubernetesRuntimeType(clientConfig)
	if err != nil {
		logrus.Error(err)
	}

	createOpts := crd.CreateOpts{
		DetectK3sRke2:  opts.DetectK3sRke2,
		UpdateCRDs:     opts.UpdateCRDs,
		K8sRuntimeType: k8sRuntimeType,
	}
	if err := crd.Create(ctx, clientConfig, createOpts); err != nil {
		return err
	}

	return controllers.Register(ctx, systemNamespace, cfg, opts)
}

func identifyKubernetesRuntimeType(clientConfig *rest.Config) (string, error) {
	client, err := clients.NewFromConfig(clientConfig, nil)
	if err != nil {
		return "", err
	}

	nodes, err := client.Core.Node().List(metav1.ListOptions{})
	if err != nil {
		logrus.Fatalf("Failed to list nodes: %v", err)
	}
	instanceTypes := make(map[string]int)
	for _, node := range nodes.Items {
		instanceType, exists := node.Labels["node.kubernetes.io/instance-type"]
		if exists {
			instanceTypes[instanceType]++
		} else {
			logrus.Debugf("Cannot find `node.kubernetes.io/instance-type` label on node `%s`", node.Name)
		}
	}

	if len(instanceTypes) == 0 {
		return "", errors.New("cannot identify k8s runtime type; no nodes in cluster have expected label")
	}

	var k8sRuntimeType string
	for instanceType := range instanceTypes {
		k8sRuntimeType = instanceType
		break
	}

	return k8sRuntimeType, nil
}
