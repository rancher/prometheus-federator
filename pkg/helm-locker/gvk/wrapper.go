package gvk

import (
	"context"

	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/relatedresource"
)

// sharedControllerToWrapper converts a SharedController to a relatedresource.ControllerWrapper
type sharedControllerToWrapper struct {
	controller.SharedController
}

// AddGenericHandler registers a generic Handler on a SharedController
func (w sharedControllerToWrapper) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	w.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

// wrapController returns a relatedresource.ControllerWrapper on top of a SharedController
func wrapController(c controller.SharedController) relatedresource.ControllerWrapper {
	return sharedControllerToWrapper{c}
}
