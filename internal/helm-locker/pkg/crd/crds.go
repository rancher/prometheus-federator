package crd

import (
	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/crd"
)

// List returns the set of CRDs that need to be generated
func Required() []crd.CRD {

	helmRelease := crd.NamespacedType("HelmRelease.helm.cattle.io/v1alpha1").
		WithSchemaFromStruct(v1alpha1.HelmRelease{}).
		WithColumn("Release Name", ".spec.release.name").
		WithColumn("Release Namespace", ".spec.release.namespace").
		WithColumn("Version", ".status.version").
		WithColumn("State", ".status.state").
		WithStatus()
	return []crd.CRD{
		helmRelease,
	}
}
