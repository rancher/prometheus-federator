package common

import (
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"
)

// User-Applied Labels
// Note: These labels are expected to be applied by users (or by Jobs, in the case of cleanup), to mark a resources as one that needs
// some special logic to be applied by the Helm Project Operator on changes

// ProjectHelmCharts

const (
	// HelmProjectOperatedCleanupLabel is a label attached to ProjectHelmCharts to facilitate cleanup; all ProjectHelmCharts
	// with this label will have their HelmCharts and HelmReleases cleaned up until the next time the Operator is deployed;
	// on redeploying the operator, this label will automatically be removed from all ProjectHelmCharts deployed in the cluster.
	HelmProjectOperatedCleanupLabel = "helm.cattle.io/helm-project-operator-cleanup"
)

// HasCleanupLabel returns whether a ProjectHelmChart has the cleanup label
func HasCleanupLabel(projectHelmChart *v1alpha1.ProjectHelmChart) bool {
	if projectHelmChart.Labels == nil {
		return false
	}
	value, shouldCleanup := projectHelmChart.Labels[HelmProjectOperatedCleanupLabel]
	return shouldCleanup && value == "true"
}

// Project Release Namespace ConfigMaps

const (
	// HelmProjectOperatorDashboardValuesConfigMapLabel is a label that identifies a ConfigMap that should be merged into status.dashboardValues when available
	// The value of this label will be the release name of the Helm chart, which will be used to identify which ProjectHelmChart's status needs to be updated.
	HelmProjectOperatorDashboardValuesConfigMapLabel = "helm.cattle.io/dashboard-values-configmap"
)

// Project Release Namespace Roles

const (
	// HelmProjectOperatorProjectHelmChartRoleLabel is a label that identifies a Role as one that needs RoleBindings to be managed by the Helm Project Operator
	// The value of this label will be the release name of the Helm chart, which will be used to identify which ProjectHelmChart's enqueue should resynchronize this.
	HelmProjectOperatorProjectHelmChartRoleLabel = "helm.cattle.io/project-helm-chart-role"

	// HelmProjectOperatorProjectHelmChartRoleAggregateFromLabel is a label that identifies which subjects should be bound to the Project Helm Chart Role
	// The value of this label will be the name of the default k8s ClusterRoles (cluster-admin, admin, edit, view). For the provided ClusterRole,
	// the operator will automatically create a RoleBinding in the Project Release Namespace binding all subjects who have that permission across all namespaces in the project
	// to the Role that contains this label. This label will only be viewed if the Role has HelmProjectOperatorProjectHelmChartRoleLabel set as well
	HelmProjectOperatorProjectHelmChartRoleAggregateFromLabel = "helm.cattle.io/project-helm-chart-role-aggregate-from"
)
