package crd

import (
	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/crd"
)

func Required() []crd.CRD {
	// project helm chart
	projectChart := crd.NamespacedType("ProjectHelmChart.helm.cattle.io/v1alpha1").
		WithSchemaFromStruct(v1alpha1.ProjectHelmChart{}).
		WithColumn("Status", ".status.status").
		WithColumn("System Namespace", ".status.systemNamespace").
		WithColumn("Release Namespace", ".status.releaseNamespace").
		WithColumn("Release Name", ".status.releaseName").
		WithColumn("Target Namespaces", ".status.targetNamespaces").
		WithStatus()

	return []crd.CRD{projectChart}
}
