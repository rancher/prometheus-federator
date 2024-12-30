package crd

import (
	v1alpha1 "github.com/rancher/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// List returns the set of CRDs that need to be generated
func Required() []crd.CRD {
	return []crd.CRD{
		newCRD(&v1alpha1.HelmRelease{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("Release Name", ".spec.release.name").
				WithColumn("Release Namespace", ".spec.release.namespace").
				WithColumn("Version", ".status.version").
				WithColumn("State", ".status.state")
		}),
	}
}

// newCRD returns the CustomResourceDefinition of an object that is customized
// according to the provided customize function
func newCRD(obj interface{}, customize func(crd.CRD) crd.CRD) crd.CRD {
	crd := crd.CRD{
		GVK: schema.GroupVersionKind{
			Group:   "helm.cattle.io",
			Version: "v1alpha1",
		},
		Status:       true,
		SchemaObject: obj,
	}
	if customize != nil {
		crd = customize(crd)
	}
	return crd
}
