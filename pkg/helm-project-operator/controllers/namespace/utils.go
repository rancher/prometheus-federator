package namespace

import (
	"fmt"

	"github.com/rancher/wrangler/pkg/apply"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// configureApplyForNamespace configures the apply to treat the provided namespace as an owner
func (h *handler) configureApplyForNamespace(namespace *corev1.Namespace) apply.Apply {
	return h.apply.
		WithOwner(namespace).
		// Why do we need the release name?
		// To ensure that we don't override the set created by another instance of the Project Operator
		// running under a different release name operating on the same project registration namespace
		WithSetID(fmt.Sprintf("%s-%s-data", namespace.Name, h.opts.ReleaseName))
}

// getProjectIDFromNamespaceLabels returns projectIDs based on the label on the project
func (h *handler) getProjectIDFromNamespaceLabels(namespace *corev1.Namespace) (string, bool) {
	if len(h.opts.ProjectLabel) == 0 {
		// nothing to do, namespaces are not project scoped
		return "", false
	}
	labels := namespace.GetLabels()
	if labels == nil {
		return "", false
	}
	projectID, namespaceInProject := labels[h.opts.ProjectLabel]
	return projectID, namespaceInProject
}

// enqueueProjectHelmChartsForNamespace simply enqueues all ProjectHelmCharts in a namespace
func (h *handler) enqueueProjectHelmChartsForNamespace(namespace *corev1.Namespace) error {
	projectHelmCharts, err := h.projectHelmChartCache.List(namespace.Name, labels.Everything())
	if err != nil {
		return err
	}
	for _, projectHelmChart := range projectHelmCharts {
		h.projectHelmCharts.Enqueue(projectHelmChart.Namespace, projectHelmChart.Name)
	}
	return nil
}
