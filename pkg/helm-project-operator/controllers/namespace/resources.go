package namespace

import (
	"fmt"
	"strings"

	common2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Note: each resource created here should have a resolver set in resolvers.go
// The only exception is namespaces since those are handled by the main controller OnChange

// getProjectRegistrationNamespace returns the namespace created on behalf of a new Project that has been identified based on
// unique values observed for all namespaces with the label h.opts.ProjectLabel
func (h *handler) getProjectRegistrationNamespace(projectID string, isOrphaned bool) *corev1.Namespace {
	if len(h.opts.ProjectLabel) == 0 {
		return nil
	}
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf(common2.ProjectRegistrationNamespaceFmt, projectID),
			Annotations: common2.GetProjectNamespaceAnnotations(projectID, h.opts.ProjectLabel, h.opts.ClusterID),
			Labels:      common2.GetProjectNamespaceLabels(projectID, h.opts.ProjectLabel, projectID, isOrphaned),
		},
	}
}

// getConfigMap returns the values.yaml and questions.yaml ConfigMap that is expected to be created in all Project Registration Namespaces
func (h *handler) getConfigMap(projectID string, namespace *corev1.Namespace) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getConfigMapName(),
			Namespace: namespace.Name,
			Labels:    common2.GetCommonLabels(projectID),
		},
		Data: map[string]string{
			"values.yaml":    h.valuesYaml,
			"questions.yaml": h.questionsYaml,
		},
	}
}

// getConfigMap name returns the name of the ConfigMap to be deployed in all Project Registration Namespaces
func (h *handler) getConfigMapName() string {
	return strings.ReplaceAll(h.opts.HelmAPIVersion, "/", ".")
}
