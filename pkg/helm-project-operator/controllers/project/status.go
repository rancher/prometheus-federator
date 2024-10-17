package project

import (
	"fmt"

	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"
)

// getCleanupStatus returns the status on seeing the cleanup label on a ProjectHelmChart
func (h *handler) getCleanupStatus(projectHelmChart *v1alpha1.ProjectHelmChart, _ v1alpha1.ProjectHelmChartStatus) v1alpha1.ProjectHelmChartStatus {
	return v1alpha1.ProjectHelmChartStatus{
		Status: "AwaitingOperatorRedeployment",
		StatusMessage: fmt.Sprintf(
			"ProjectHelmChart was marked with label %s=true, which indicates that the resource should be cleaned up "+
				"until the Project Operator that responds to ProjectHelmCharts in %s with spec.helmApiVersion=%s "+
				"is redeployed onto the cluster. On redeployment, this label will automatically be removed by the operator.",
			common.HelmProjectOperatedCleanupLabel, projectHelmChart.Namespace, projectHelmChart.Spec.HelmAPIVersion,
		),
	}
}

// getUnableToCreateHelmReleaseStatus returns the status on seeing a conflicting ProjectHelmChart already tracking the desired Helm release
func (h *handler) getUnableToCreateHelmReleaseStatus(projectHelmChart *v1alpha1.ProjectHelmChart, _ v1alpha1.ProjectHelmChartStatus, err error) v1alpha1.ProjectHelmChartStatus {
	releaseNamespace, releaseName := h.getReleaseNamespaceAndName(projectHelmChart)
	return v1alpha1.ProjectHelmChartStatus{
		Status: "UnableToCreateHelmRelease",
		StatusMessage: fmt.Sprintf(
			"Unable to create a release (%s/%s) for ProjectHelmChart: %s",
			releaseName, releaseNamespace, err,
		),
	}
}

// getNoTargetNamespacesStatus returns the status on seeing that a ProjectHelmChart's projectNamespaceSelector (or
// the Project Registration Namespace's namespaceSelector) targets no namespaces
func (h *handler) getNoTargetNamespacesStatus(_ *v1alpha1.ProjectHelmChart, _ v1alpha1.ProjectHelmChartStatus) v1alpha1.ProjectHelmChartStatus {
	return v1alpha1.ProjectHelmChartStatus{
		Status:        "NoTargetProjectNamespaces",
		StatusMessage: "There are no project namespaces to deploy a ProjectHelmChart.",
	}
}

// getValuesParseErrorStatus returns the status on encountering an error with parsing the provided contents of spec.values on the ProjectHelmChart
func (h *handler) getValuesParseErrorStatus(_ *v1alpha1.ProjectHelmChart, projectHelmChartStatus v1alpha1.ProjectHelmChartStatus, err error) v1alpha1.ProjectHelmChartStatus {
	// retain existing status if possible
	projectHelmChartStatus.Status = "UnableToParseValues"
	projectHelmChartStatus.StatusMessage = fmt.Sprintf("Unable to convert provided spec.values into valid configuration of ProjectHelmChart: %s", err)
	return projectHelmChartStatus
}

// getWaitingForDashboardValuesStatus returns the transitionary status that occurs after deploying a Helm chart but before a dashboard configmap is created
// If a ProjectHelmChart is stuck in this status, it is likely either an error on the Operator for not creating this ConfigMap or there might be an issue
// with the underlying Job ran by the child HelmChart resource created on this ProjectHelmChart's behalf
func (h *handler) getWaitingForDashboardValuesStatus(_ *v1alpha1.ProjectHelmChart, projectHelmChartStatus v1alpha1.ProjectHelmChartStatus) v1alpha1.ProjectHelmChartStatus {
	// retain existing status
	projectHelmChartStatus.Status = "WaitingForDashboardValues"
	projectHelmChartStatus.StatusMessage = "Waiting for status.dashboardValues content to be provided by the deployed Helm release, but HelmChart and HelmRelease should be deployed."
	projectHelmChartStatus.DashboardValues = nil
	return projectHelmChartStatus
}

// getDeployedStatus returns the status that indicates the ProjectHelmChart is successfully deployed
func (h *handler) getDeployedStatus(_ *v1alpha1.ProjectHelmChart, projectHelmChartStatus v1alpha1.ProjectHelmChartStatus) v1alpha1.ProjectHelmChartStatus {
	// retain existing status
	projectHelmChartStatus.Status = "Deployed"
	projectHelmChartStatus.StatusMessage = "ProjectHelmChart has been successfully deployed!"
	return projectHelmChartStatus
}
