package hack

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/name"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	nodev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/sirupsen/logrus"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nodeLabel = "node.kubernetes.io/instance-type"
)

type CRDManagementConfig struct {
	options common.RuntimeOptions
}

func CRDName(crd crd.CRD) string {
	return crd.GVK.GroupVersion().WithKind(strings.ToLower(name.GuessPluralName(crd.GVK.Kind))).GroupKind().String()
}

func FilterMissingCRDs(client apiextensionv1.CustomResourceDefinitionInterface, expectedCRDs []crd.CRD) ([]crd.CRD, error) {
	ret := []crd.CRD{}
	for _, currentCRD := range expectedCRDs {
		crdName := CRDName(currentCRD)
		// try to get the given CRD just to check for error, verifying if it exists
		foundCRD, err := client.Get(context.TODO(), crdName, metav1.GetOptions{})
		if err == nil {
			storedVersions := foundCRD.Status.StoredVersions
			logrus.Debugf(
				"Found `%s` at versions `%s`, expecting version `%s`",
				crdName,
				strings.Join(storedVersions, ","),
				currentCRD.GVK.Version,
			)
			if slices.Contains(storedVersions, currentCRD.GVK.Version) {
				logrus.Debugf("Installing `%s` will be skipped; a suitible version exists on the cluster", crdName)
				continue
			}
			logrus.Debugf("No suitable version `%s` found for `%s`, queuing it for install", currentCRD.GVK.Version, crdName)
			ret = append(ret, currentCRD)
		} else if !apierrors.IsNotFound(err) {
			return nil, err
		} else {
			logrus.Debugf("Did not find `%s` on the cluster, it will be installed", crdName)
			ret = append(ret, currentCRD)
		}
	}
	return ret, nil
}

// IdentifyKubernetesRuntimeType provides the k8s runtime used on nodes in the cluster.
// Deprecated: This feature is a stop gap not expected to be maintained long-term.
// A more robust solution should be implemented, either in the `helm-controller` or in `wrangler` CRD frameworks.
func IdentifyKubernetesRuntimeType(client nodev1.NodeInterface) (string, error) {
	nodes, err := client.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("Failed to list nodes: %v", err)
		return "", err
	}
	instanceTypes := make(map[string]int)
	for _, node := range nodes.Items {
		instanceType, exists := node.Labels[nodeLabel]
		if exists {
			instanceTypes[instanceType]++
		} else {
			logrus.Debugf("Cannot find `%s` label on node `%s`", nodeLabel, node.Name)
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
