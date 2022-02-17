package v1alpha1

import (
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rancher/wrangler/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IdentifiedNamespaces = "IdentifiedNamespaces"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProjectSpec   `json:"spec"`
	Status            ProjectStatus `json:"status"`
}

type ProjectSpec struct {
	Selector *metav1.LabelSelector `json:"selector"`

	Prometheus   PrometheusSpec   `json:"prometheus"`
	Alertmanager AlertmanagerSpec `json:"alertmanager"`
	Grafana      GrafanaSpec      `json:"grafana"`
}

type PrometheusSpec struct {
	v1.PrometheusSpec
}

type AlertmanagerSpec struct {
	Enabled bool `json:"enabled"`
	v1.AlertmanagerSpec
}

type GrafanaSpec struct {
	Enabled bool `json:"enabled"`
	v1.AlertmanagerSpec
}

type ProjectStatus struct {
	ClusterPrometheus string   `json:"clusterPrometheus"`
	ProjectNamespace  string   `json:"projectNamespace"`
	Namespaces        []string `json:"namespaces"`

	GrafanaDeployed      bool                                `json:"grafanaDeployed"`
	PrometheusDeployed   bool                                `json:"prometheusDeployed"`
	AlertmanagerDeployed bool                                `json:"alertmanagerDeployed"`
	Conditions           []genericcondition.GenericCondition `json:"conditions,omitempty"`
}
