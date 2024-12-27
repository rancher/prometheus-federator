package objectset

import (
	"context"
	"time"

	"github.com/rancher/helm-locker/pkg/gvk"
	"github.com/rancher/helm-locker/pkg/informerfactory"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/apply"
	"github.com/rancher/wrangler/v3/pkg/start"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/workqueue"
)

// NewLockableRegister returns a starter that starts an ObjectSetController listening to events on ObjectSetStates
// and a LockableRegister that allows you to register new states for ObjectSets in memory
func NewLockableRegister(name string, apply apply.Apply, scf controller.SharedControllerFactory, discovery discovery.DiscoveryInterface, opts *controller.Options) (start.Starter, LockableRegister, *controller.SharedHandler) {
	// Define a new cache
	apply = apply.WithCacheTypeFactory(informerfactory.New(scf))

	handler := handler{
		apply:         apply,
		gvkLister:     gvk.NewLister(discovery),
		sharedHandler: &controller.SharedHandler{},
	}

	lockableObjectSetRegister, objectSetCache := newLockableObjectSetRegisterAndCache(scf, handler.OnRemove)

	handler.locker = lockableObjectSetRegister

	startCache := func(ctx context.Context) error {
		go objectSetCache.Run(ctx.Done())
		return nil
	}

	// Define a new controller that responds to events from the cache
	objectSetController := controller.New(name, objectSetCache, startCache, &handler, applyDefaultOptions(opts))

	return wrapStarter(objectSetController), lockableObjectSetRegister, handler.sharedHandler
}

// applyDefaultOptions applies default controller options if none are provided
func applyDefaultOptions(opts *controller.Options) *controller.Options {
	var newOpts controller.Options
	if opts != nil {
		newOpts = *opts
	}
	if newOpts.RateLimiter == nil {
		newOpts.RateLimiter = workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemFastSlowRateLimiter(time.Millisecond, 2*time.Minute, 30),
			workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 30*time.Second),
		)
	}
	return &newOpts
}
