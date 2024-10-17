package project

import (
	"fmt"

	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"
)

// getProjectID returns the projectID tied to this ProjectHelmChart
func (h *handler) getProjectID(projectHelmChart *v1alpha1.ProjectHelmChart) (string, error) {
	if len(h.opts.ProjectLabel) == 0 {
		// use the projectHelmChart's name as the projectID
		return projectHelmChart.Name, nil
	}
	projectRegistrationNamespace, err := h.namespaceCache.Get(projectHelmChart.Namespace)
	if err != nil {
		return "", fmt.Errorf("unable to parse projectID for projectHelmChart %s/%s: %s", projectHelmChart.Namespace, projectHelmChart.Name, err)
	}
	projectID, ok := projectRegistrationNamespace.Labels[h.opts.ProjectLabel]
	if !ok {
		return "", nil
	}
	return projectID, nil
}

// getProjectNamespaceSelector returns the projectNamespaceSelector tied to this ProjectHelmChart
func (h *handler) getProjectNamespaceSelector(projectHelmChart *v1alpha1.ProjectHelmChart, projectID string) map[string]interface{} {
	if len(h.opts.ProjectLabel) == 0 {
		// Use the projectHelmChart selector as the namespaceSelector
		if projectHelmChart.Spec.ProjectNamespaceSelector == nil {
			return map[string]interface{}{}
		}
		return map[string]interface{}{
			"matchLabels":      projectHelmChart.Spec.ProjectNamespaceSelector.MatchLabels,
			"matchExpressions": projectHelmChart.Spec.ProjectNamespaceSelector.MatchExpressions,
		}
	}
	if len(h.opts.ProjectReleaseLabelValue) == 0 {
		// Release namespace is not created, so use namespaceSelector provided tied to projectID
		return map[string]interface{}{
			"matchLabels": map[string]string{
				h.opts.ProjectLabel: projectID,
			},
		}
	}
	// use the HelmProjectOperated label
	return map[string]interface{}{
		"matchLabels": map[string]string{
			common.HelmProjectOperatorProjectLabel: projectID,
		},
	}
}

// getReleaseNamespaceAndName returns the name of the Project Release namespace and the name of the Helm Release
// that will be deployed into the Project Release namespace on behalf of the ProjectHelmChart
func (h *handler) getReleaseNamespaceAndName(projectHelmChart *v1alpha1.ProjectHelmChart) (string, string) {
	projectReleaseName := fmt.Sprintf("%s-%s", projectHelmChart.Name, h.opts.ReleaseName)
	if h.opts.Singleton {
		// This changes the naming scheme of the deployed resources such that only one can every be created per namespace
		projectReleaseName = fmt.Sprintf("%s-%s", projectHelmChart.Namespace, h.opts.ReleaseName)
	}
	if len(h.opts.ProjectLabel) == 0 || len(h.opts.ProjectReleaseLabelValue) == 0 {
		// Underlying Helm releases will be created in the namespace where the ProjectHelmChart is registered (project registration namespace)
		// The project registration namespace will either be the system namespace or auto-generated namespaces depending on the user values provided
		return projectHelmChart.Namespace, projectReleaseName
	}
	// Underlying Helm releases will be created in dedicated project release namespaces
	return projectReleaseName, projectReleaseName
}
