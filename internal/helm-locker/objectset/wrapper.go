package objectset

import (
	"context"

	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/start"
)

// controllerToStarterWrapper wraps the generic controller.Controller interface with a dummy call for Sync
type controllerToStarterWrapper struct {
	controller.Controller
}

// Sync does a noop
func (w controllerToStarterWrapper) Sync(_ context.Context) error {
	return nil
}

// wrapStarter returns a start.Starter around a controller.Controller
func wrapStarter(c controller.Controller) start.Starter {
	return controllerToStarterWrapper{c}
}
