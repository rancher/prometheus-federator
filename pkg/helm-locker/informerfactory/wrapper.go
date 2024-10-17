package informerfactory

import (
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/apply"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// New wraps the provided SharedControllerFactory to satisfy the apply.InformerFactory interface
func New(scf controller.SharedControllerFactory) apply.InformerFactory {
	return informerFactoryWrapper{
		SharedControllerFactory: scf,
	}
}

// informerFactoryWrapper satisfies the apply.InformerFactory interface
type informerFactoryWrapper struct {
	controller.SharedControllerFactory
}

// Get returns a cache.SharedIndexInformer for a given GVK from the SharedControllerFactory
func (w informerFactoryWrapper) Get(gvk schema.GroupVersionKind, _ schema.GroupVersionResource) (cache.SharedIndexInformer, error) {
	controller, err := w.ForKind(gvk)
	if err != nil {
		return nil, nil
	}
	return controller.Informer(), nil
}
