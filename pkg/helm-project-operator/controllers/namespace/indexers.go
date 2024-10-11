package namespace

import (
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
)

const (
	// NamespacesByProjectExcludingRegistrationID is an index mapping namespaces to project that they belong into
	// The index will omit any namespaces considered to be the Project Registration namespace or a system namespace
	NamespacesByProjectExcludingRegistrationID = "helm.cattle.io/namespaces-by-project-id-excluding-registration"
)

// initIndexers initializes indexers that allow for more efficient computations on related resources without relying on additional
// calls to be made to the Kubernetes API by referencing the cache instead
func (h *handler) initIndexers() {
	h.namespaceCache.AddIndexer(NamespacesByProjectExcludingRegistrationID, h.namespaceToProjectIDExcludingRegistration)
}

func (h *handler) namespaceToProjectIDExcludingRegistration(namespace *corev1.Namespace) ([]string, error) {
	if namespace == nil {
		return nil, nil
	}
	if h.isSystemNamespace(namespace) {
		return nil, nil
	}
	if h.isProjectRegistrationNamespace(namespace) {
		return nil, nil
	}
	if namespace.Labels[common.HelmProjectOperatedLabel] == "true" {
		// always ignore Helm Project Operated namespaces since those are only
		// to be scoped to namespaces that are project registration namespaces
		return nil, nil
	}
	projectID, inProject := h.getProjectIDFromNamespaceLabels(namespace)
	if !inProject {
		// nothing to do
		return nil, nil
	}
	return []string{projectID}, nil
}
