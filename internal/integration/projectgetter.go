package integration

import (
	"slices"

	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	corev1 "k8s.io/api/core/v1"
)

type embedProjectGetter struct {
	projRegistration []string
	system           []string
	targetNamespaces map[string][]string
}

func (e *embedProjectGetter) IsProjectRegistrationNamespace(namespace *corev1.Namespace) bool {
	if namespace == nil {
		return false
	}
	return slices.Contains(e.projRegistration, namespace.Name)
}

func (e *embedProjectGetter) IsSystemNamespace(namespace *corev1.Namespace) bool {
	if namespace == nil {
		return false
	}
	return slices.Contains(e.system, namespace.Name)
}

// GetTargetProjectNamespaces returns the list of namespaces that should be targeted for a given ProjectHelmChart
// Any namespace returned by this should not be a project registration namespace or a system namespace
func (e *embedProjectGetter) GetTargetProjectNamespaces(projectHelmChart *v1alpha1.ProjectHelmChart) ([]string, error) {
	if projectHelmChart == nil {
		return []string{}, nil
	}
	vals, ok := e.targetNamespaces[projectHelmChart.Name]
	if !ok {
		return []string{}, nil
	}
	return vals, nil
}

var _ namespace.ProjectGetter = (*embedProjectGetter)(nil)

func projectGetter(
	projRegistration []string,
	system []string,
	targetNamespaces map[string][]string,
) namespace.ProjectGetter {

	return &embedProjectGetter{
		projRegistration: projRegistration,
		system:           system,
		targetNamespaces: targetNamespaces,
	}
}
