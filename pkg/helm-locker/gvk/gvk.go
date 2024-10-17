package gvk

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resolver is a relatedresource.Resolver that can work on multiple GVKs
type Resolver func(gvk schema.GroupVersionKind, namespace, name string, _ runtime.Object) ([]relatedresource.Key, error)

// ForGVK returns the relatedresource.Resolver for a particular GVK
func (r Resolver) ForGVK(gvk schema.GroupVersionKind) relatedresource.Resolver {
	return func(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
		return r(gvk, namespace, name, obj)
	}
}

// Watcher starts controllers for one or more GVKs using the provided SharedControllerFactory
// After starting a GVK controller, it will register a relatedresource.Watch using the provided
// relatedresource.Enqueuer and Resolver
type Watcher interface {
	// Start will run all the watchers that have been registered thus far and deferred from starting
	Start(ctx context.Context, workers int) error
	// Watch will start a new watcher for a particular GVK; if the Watcher has not started yet,
	// watching will be deferred till the first Start call is made.
	Watch(gvk schema.GroupVersionKind) error
}

// NewWatcher returns an object that satisfies the Watcher interface
func NewWatcher(scf controller.SharedControllerFactory, gvkResolver Resolver, enqueuer relatedresource.Enqueuer) Watcher {
	return &watcher{
		scf:         scf,
		gvkResolver: gvkResolver,
		enqueuer:    enqueuer,

		gvkRegistered: make(map[schema.GroupVersionKind]bool),
		gvkStarted:    make(map[schema.GroupVersionKind]bool),
	}
}

// watcher is a Watcher based on a provided resolver and enqueuer
type watcher struct {

	// scf is the controller.SharedControllerFactory to use to generate controllers from
	scf controller.SharedControllerFactory

	// gvkResolver is the Resolver that is used to register the relatedresource.Watch
	gvkResolver Resolver

	// enqueuer is the relatedresource.Enqueuer that is used to register the relatedresource.Watch
	enqueuer relatedresource.Enqueuer

	// gvkRegistered is the list of all gvks that have been registered for Watch
	// note: the associated gvkControllers will not be started if this Watcher has not been started yet
	gvkRegistered map[schema.GroupVersionKind]bool
	// gvkStarted is the list of all gvks that have already started watching and triggering enqueues
	gvkStarted map[schema.GroupVersionKind]bool

	// started is whether the Watcher has started actually registering relatedresource.Watch
	started bool
	// controllerCtx is the context provided on start that all watchers will use
	controllerCtx context.Context
	// controllerWorkers is the number of worker threads each watcher should use to process resources
	controllerWorkers int

	// lock ensures concurrent calls to Watch and Start happen atomically
	lock sync.RWMutex
}

// Watch begins watching a GVK or defers its start for after Start is called
func (w *watcher) Watch(gvk schema.GroupVersionKind) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.gvkRegistered[gvk] = true
	return w.startGVK(gvk)
}

// Start begins watching all registered GVKs
func (w *watcher) Start(ctx context.Context, workers int) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.started = true
	w.controllerCtx = ctx
	w.controllerWorkers = workers
	var multierr error
	for gvk := range w.gvkRegistered {
		if err := w.startGVK(gvk); err != nil {
			multierr = multierror.Append(multierr, err)
		}
	}
	return multierr
}

// startGVK starts watching a particular GVK if the Watcher has been started
func (w *watcher) startGVK(gvk schema.GroupVersionKind) error {
	if !w.started {
		return nil
	}
	if _, ok := w.gvkStarted[gvk]; ok {
		// gvk was already started
		return nil
	}
	gvkController, err := w.scf.ForKind(gvk)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s Watcher", gvk)
	logrus.Infof("Starting %s", name)

	// NOTE: The order here (namely, calling relatedresource.Watch before gvkController.Start) is important.
	//
	// By default, the controller returned by a shared controller factory is a deferred controller
	// that won't populate the actual underlying controller until at least one function is called on
	// the controller (e.g. Enqueue, EnqueueAfter, EnqueueKey, Informer, or RegisterHandler)
	//
	// Therefore, running Start on an empty controller will result in the controller never registering
	// the relatedresource.Watch we provide here, since the underlying informer is nil.

	relatedresource.Watch(
		w.controllerCtx,
		name,
		w.gvkResolver.ForGVK(gvk),
		w.enqueuer,
		wrapController(gvkController),
	)

	if err := gvkController.Start(w.controllerCtx, w.controllerWorkers); err != nil {
		return err
	}
	w.gvkStarted[gvk] = true
	return nil
}
