package common

import (
	"fmt"
	"strings"
)

// Operator Labels
// Note: These labels are automatically applied by the operator to mark resources that are created for a given ProjectHelmChart and Project Operator

// Common

const (
	// HelmProjectOperatedLabel marks all HelmCharts, HelmReleases, and namespaces created by this operator
	HelmProjectOperatedLabel = "helm.cattle.io/helm-project-operated"

	// HelmProjectOperatorProjectLabel is applied to the Project Registration Namespace, the ProjectReleaseNamespace, and
	// (only if both ProjectLabel and ProjectReleaseLabelValue are provided) to all Project namespaces
	//
	// If ProjectLabel and ProjectReleaseLabelValue are supplied, this label will be supplied to the global.cattle.projectNamespaceSelector
	// to identify all namespaces tied to a given project
	HelmProjectOperatorProjectLabel = "helm.cattle.io/projectId"
)

// HasHelmProjectOperatedLabel returns whether a ProjectHelmChart has the Helm Project Operated label
func HasHelmProjectOperatedLabel(labels map[string]string) bool {
	if labels == nil {
		return false
	}
	_, ok := labels[HelmProjectOperatedLabel]
	return ok
}

// GetCommonLabels returns all common labels added to all generated resources
func GetCommonLabels(projectID string) map[string]string {
	labels := map[string]string{
		HelmProjectOperatedLabel: "true",
	}
	if len(projectID) != 0 {
		labels[HelmProjectOperatorProjectLabel] = projectID
	}
	return labels
}

// Project Namespaces

const (
	// HelmProjectOperatedNamespaceOrphanedLabel marks all auto-generated namespaces that no longer have resources tracked
	// by this operator; if a namespace has this label, it is safe to delete
	HelmProjectOperatedNamespaceOrphanedLabel = "helm.cattle.io/helm-project-operator-orphaned"
)

// GetProjectNamespaceLabels returns the labels to be added to all Project Namespaces
func GetProjectNamespaceLabels(projectID, projectLabel, projectLabelValue string, isOrphaned bool) map[string]string {
	labels := GetCommonLabels(projectID)
	if isOrphaned {
		labels[HelmProjectOperatedNamespaceOrphanedLabel] = "true"
	}
	labels[projectLabel] = projectLabelValue
	return labels
}

// GetProjectNamespaceAnnotations returns the annotations to be added to all Project Namespaces
// Note: annotations allow integration with Rancher Projects since they handle importing namespaces into Projects
func GetProjectNamespaceAnnotations(projectID, projectLabel, clusterID string) map[string]string {
	projectIDWithClusterID := projectID
	if len(clusterID) > 0 {
		projectIDWithClusterID = fmt.Sprintf("%s:%s", clusterID, projectID)
	}
	return map[string]string{
		projectLabel: projectIDWithClusterID,
	}
}

// Helm Resources (HelmCharts and HelmReleases)

const (
	// HelmProjectOperatorHelmAPIVersionLabel is a label that identifies the HelmAPIVersion that a HelmChart or HelmRelease is tied to
	// This is used to identify whether a HelmChart or HelmRelease should be deleted from the cluster on uninstall
	HelmProjectOperatorHelmAPIVersionLabel = "helm.cattle.io/helm-api-version"
)

// GetHelmResourceLabels returns the labels to be added to all generated Helm resources (HelmCharts, HelmReleases)
func GetHelmResourceLabels(projectID, helmAPIVersion string) map[string]string {
	labels := GetCommonLabels(projectID)
	labels[HelmProjectOperatorHelmAPIVersionLabel] = strings.SplitN(helmAPIVersion, "/", 2)[0]
	return labels
}

// RoleBindings (created for Default K8s ClusterRole RBAC aggregation)

const (
	// HelmProjectOperatorProjectHelmChartRoleBindingLabel is a label that identifies a RoleBinding as one that has been created in response to a ProjectHelmChart role
	// The value of this label will be the release name of the Helm chart, which will be used to identify which ProjectHelmChart's enqueue should resynchronize this.
	HelmProjectOperatorProjectHelmChartRoleBindingLabel = "helm.cattle.io/project-helm-chart-role-binding"
)
