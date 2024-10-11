package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectHelmChart specifies a managed Helm chart that should be deployed for a "Project" (defined as any set
// of namespaces that can be targeted by a label selector) and be updated automatically on changing definitions
// of that project (e.g. namespaces added or removed). It is a parent object that creates HelmCharts and HelmReleases
// under the hood via wrangler.Apply and relatedresource.Watch
type ProjectHelmChart struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProjectHelmChartSpec   `json:"spec"`
	Status            ProjectHelmChartStatus `json:"status"`
}

// ProjectHelmChartSpec defines the spec of a ProjectHelmChart
type ProjectHelmChartSpec struct {
	// HelmAPIVersion identifies whether a particular rendition of the Helm Project Operator
	// should watch ProjectHelmChart of this type. e.g. monitoring.cattle.io/v1alpha1 is watched by Prometheus Federator
	HelmAPIVersion string `json:"helmApiVersion"`

	// ProjectNamespaceSelector is a namespaceSelector that identifies the project this underlying chart should be targeting
	// If a project label is provided as part of the Operator's runtime options, this field will be ignored since ProjectHelmCharts
	// will be created in dedicated project namespaces with a pre-defined project namespace selector
	ProjectNamespaceSelector *metav1.LabelSelector `json:"projectNamespaceSelector"`

	// Values is a generic map (e.g. generic yaml) representing the values.yaml used to configure the underlying Helm chart that
	// will be deployed for this
	Values GenericMap `json:"values"`
}

type ProjectHelmChartStatus struct {
	// DashboardValues are values provided to the ProjectHelmChart from ConfigMaps in the Project Release namespace
	// tagged with 'helm.cattle.io/dashboard-values-configmap': '{{ .Release.Name }}'
	DashboardValues GenericMap `json:"dashboardValues"`

	// Status is the current status of this ProjectHelmChart
	// Please see pkg/controllers/project/status.go for possible states
	Status string `json:"status"`

	// StatusMessage is a detailed message explaining the current status of the ProjectHelmChart
	// Please see pkg/controllers/project/status.go for possible state messages
	StatusMessage string `json:"statusMessage"`

	// SystemNamespace is the namespace where HelmCharts and HelmReleases will be deployed
	SystemNamespace string `json:"systemNamespace"`

	// ReleaseNamespace is the namespace where the underlying Helm chart will be deployed
	// Also known as the Project Release Namespace
	ReleaseNamespace string `json:"releaseNamespace"`

	// ReleaseName is the name of the Helm Release contained in the Project Release Namespace
	ReleaseName string `json:"releaseName"`

	// TargetNamespaces are the current set of namespaces targeted by the namespaceSelector
	// that this ProjectHelmChart was configured with. As noted above, this will correspond
	// to the Project Registration Namespace's selector if project label is provided
	TargetNamespaces []string `json:"targetNamespaces"`
}
