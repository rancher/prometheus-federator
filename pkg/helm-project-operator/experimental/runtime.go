package experimental

import (
	"errors"

	"github.com/rancher/wrangler/pkg/clients"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// IdentifyKubernetesRuntimeType provides the k8s runtime used on nodes in the cluster.
// Deprecated: This feature is a stop gap not expected to be maintained long-term.
// A more robust solution should be implemented, either in the `helm-controller` or in `wrangler` CRD frameworks.
func IdentifyKubernetesRuntimeType(clientConfig *rest.Config) (string, error) {
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
