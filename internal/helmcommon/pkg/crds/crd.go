package crds

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/crd"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

type CRDTrackerOptions struct {
	v1beta1    bool
	customizeF func(crd.CRD) crd.CRD
}

func (o *CRDTrackerOptions) Apply(opts ...CRDTrackerOption) {
	for _, opt := range opts {
		opt(o)
	}
}

type CRDTrackerOption func(o *CRDTrackerOptions)

func WithV1beta1(enable bool) CRDTrackerOption {
	return func(o *CRDTrackerOptions) {
		o.v1beta1 = enable
	}
}

func WithCustomizeF(customizeF func(crd.CRD) crd.CRD) CRDTrackerOption {
	return func(o *CRDTrackerOptions) {
		o.customizeF = customizeF
	}
}

type CRDTracker interface {
	Objects() (ret []runtime.Object, err error)
	List() []crd.CRD
}

func NewCRDTracker(crds []crd.CRD, opts ...CRDTrackerOption) CRDTracker {
	o := CRDTrackerOptions{
		v1beta1:    false,
		customizeF: nil,
	}

	o.Apply(opts...)

	if len(crds) == 0 {
		panic("CRDTracker must be configured to have more than one CRD")
	}

	if o.customizeF == nil {
		return &crdTracker{
			crds:              crds,
			CRDTrackerOptions: o,
		}
	}

	customizedCRDs := make([]crd.CRD, len(crds))
	for i := range crds {
		customizedCRDs[i] = o.customizeF(crds[i])
	}

	return &crdTracker{
		crds:              customizedCRDs,
		CRDTrackerOptions: o,
	}
}

var _ CRDTracker = (*crdTracker)(nil)

type crdTracker struct {
	CRDTrackerOptions
	crds []crd.CRD
}

func (c *crdTracker) Objects() (ret []runtime.Object, err error) {
	for _, crdDef := range c.crds {
		if c.v1beta1 {
			crd, err := crdDef.ToCustomResourceDefinitionV1Beta1()
			if err != nil {
				return nil, err
			}
			ret = append(ret, crd)
		} else {
			crd, err := crdDef.ToCustomResourceDefinition()
			if err != nil {
				return nil, err
			}
			ret = append(ret, crd)
		}
	}
	return
}

func (c *crdTracker) List() []crd.CRD {
	return c.crds
}

func CreateFrom(ctx context.Context, cfg *rest.Config, crds []crd.CRD) error {
	factory, err := crd.NewFactoryFromClient(cfg)
	if err != nil {
		return err
	}
	return factory.BatchCreateCRDs(ctx, crds...).BatchWait()
}

func StructToCRD(namespacedType string, obj any, customize func(crd.CRD) crd.CRD) crd.CRD {
	newCrd := crd.NamespacedType(namespacedType).
		WithSchemaFromStruct(obj).
		WithStatus()

	if customize != nil {
		newCrd = customize(newCrd)
	}
	return newCrd
}
