package applier

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

var (
	defaultRateLimiter = workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemFastSlowRateLimiter(time.Millisecond, 2*time.Minute, 30),
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 30*time.Second),
	)
)

// ApplyFunc is a func that needs to be applied on seeing a particular key be passed to an Applyinator
type ApplyFunc func(key string) error

// Options are options that can be specified to configure a desired Applyinator
type Options struct {
	RateLimiter workqueue.RateLimiter
}

// Applyinator is an interface that eventually ensures that a requested action, identified by some key,
// is applied. Any object that implements Applyinator should provide the same guarantees as the
// k8s.io/client-go/util/workqueue implementation, namely:
//
// * Fair: items processed in the order in which they are added.
// * Stingy: a single item will not be processed multiple times concurrently,
// and if an item is added multiple times before it can be processed, it
// will only be processed once.
// * Multiple consumers and producers. In particular, it is allowed for an
// item to be reenqueued while it is being processed.
type Applyinator interface {
	Apply(key string)
	Run(ctx context.Context, workers int)
}

// NewApplyinator allows you to register a function that applies an action based on whether a particular
// key is enqueued via a call to Apply. It implements k8s.io/client-go/util/workqueue under the hood, which
// allows us to ensure that the apply function is called with the following guarantees (provided by workqueues):
//
// * Fair: items processed in the order in which they are added.
// * Stingy: a single item will not be processed multiple times concurrently,
// and if an item is added multiple times before it can be processed, it
// will only be processed once.
// * Multiple consumers and producers. In particular, it is allowed for an
// item to be reenqueued while it is being processed.
func NewApplyinator(name string, applyFunc ApplyFunc, opts *Options) Applyinator {
	opts = applyDefaultOptions(opts)
	return &applyinator{
		workqueue: workqueue.NewNamedRateLimitingQueue(opts.RateLimiter, name),
		apply:     applyFunc,
	}
}

func applyDefaultOptions(opts *Options) *Options {
	var newOpts Options
	if opts != nil {
		newOpts = *opts
	}
	if newOpts.RateLimiter == nil {
		newOpts.RateLimiter = defaultRateLimiter
		logrus.Debug("No rate limiter supplied, using default rate limiter.")
	}
	return &newOpts
}

type applyinator struct {
	workqueue workqueue.RateLimitingInterface
	apply     ApplyFunc
}

// Apply triggers the Applyinator to run the provided apply func on the given key
// whenever the workqueue processes the next item
func (a *applyinator) Apply(key string) {
	a.workqueue.Add(key)
}

// Run allows the applyinator to start processing items added to its workqueue
func (a *applyinator) Run(ctx context.Context, workers int) {

	logrus.Debugf("Adding items to applyinator work queue. Workers: %d", workers)
	go func() {
		<-ctx.Done()
		a.workqueue.ShutDown()
	}()
	for i := 0; i < workers; i++ {
		go wait.Until(a.runWorker, time.Second, ctx.Done())
	}
}

func (a *applyinator) runWorker() {
	for a.processNextWorkItem() {
	}
}

func (a *applyinator) processNextWorkItem() bool {
	obj, shutdown := a.workqueue.Get()

	if shutdown {
		logrus.Debug("ProcessNextWorkItem called during shutdown. Exiting function.")
		return false
	}

	if err := a.processSingleItem(obj); err != nil {
		if !strings.Contains(err.Error(), "please apply your changes to the latest version and try again") {
			logrus.Errorf("%v", err)
		}
		return true
	}

	return true
}

func (a *applyinator) processSingleItem(obj interface{}) error {
	var (
		key string
		ok  bool
	)

	defer a.workqueue.Done(obj)

	if key, ok = obj.(string); !ok {
		a.workqueue.Forget(obj)
		logrus.Errorf("expected string in workqueue but got %#v", obj)
		return nil
	}
	if err := a.apply(key); err != nil {
		a.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
	}

	logrus.Debugf("Call to processSingleItem was successful for key: %s", key)
	a.workqueue.Forget(obj)
	return nil
}
