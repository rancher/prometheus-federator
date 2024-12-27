package crd

import (
	"context"
	"io"
	"os"
	"path/filepath"

	v1alpha1 "github.com/rancher/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/yaml"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

// WriteFile writes CRDs to the path specified
func WriteFile(filename string) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return Print(f)
}

// Print prints CRDs to out
func Print(out io.Writer) error {
	obj, err := Objects(false)
	if err != nil {
		return err
	}
	data, err := yaml.Export(obj...)
	if err != nil {
		return err
	}

	_, err = out.Write(data)
	return err
}

// Objects returns runtime.Objects for every CRD
func Objects(v1beta1 bool) (result []runtime.Object, err error) {
	for _, crdDef := range List() {
		if v1beta1 {
			crd, err := crdDef.ToCustomResourceDefinitionV1Beta1()
			if err != nil {
				return nil, err
			}
			result = append(result, crd)
		} else {
			crd, err := crdDef.ToCustomResourceDefinition()
			if err != nil {
				return nil, err
			}
			result = append(result, crd)
		}
	}
	return
}

// List returns the set of CRDs that need to be generated
func List() []crd.CRD {
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

// Create creates the necessary CRDs on starting this program onto the target cluster
func Create(ctx context.Context, cfg *rest.Config) error {
	factory, err := crd.NewFactoryFromClient(cfg)
	if err != nil {
		return err
	}

	return factory.BatchCreateCRDs(ctx, List()...).BatchWait()
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
