package project

import (
	v1alpha2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"
)

// getValues returns the values.yaml that should be applied for this ProjectHelmChart after processing default and required overrides
func (h *handler) getValues(projectHelmChart *v1alpha2.ProjectHelmChart, projectID string, targetProjectNamespaces []string) v1alpha2.GenericMap {
	// default values that are set if the user does not provide them
	values := map[string]interface{}{
		"global": map[string]interface{}{
			"cattle": map[string]interface{}{
				"systemDefaultRegistry": h.opts.SystemDefaultRegistry,
				"url":                   h.opts.CattleURL,
			},
		},
	}

	// overlay provided values, which will override the above values if provided
	values = MergeMaps(values, projectHelmChart.Spec.Values)

	// overlay operator provided values overrides, which will override the above values even if provided
	values = MergeMaps(values, h.valuesOverride)

	// required project-based values that must be set even if user tries to override them
	requiredOverrides := map[string]interface{}{
		"global": map[string]interface{}{
			"cattle": map[string]interface{}{
				"clusterId":                h.opts.ClusterID,
				"projectNamespaces":        targetProjectNamespaces,
				"projectID":                projectID,
				"releaseProjectID":         h.opts.ProjectReleaseLabelValue,
				"projectNamespaceSelector": h.getProjectNamespaceSelector(projectHelmChart, projectID),
			},
		},
	}
	// overlay required values, which will override the above values even if provided
	values = MergeMaps(values, requiredOverrides)

	return values
}
