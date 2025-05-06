//go:build integration

package test

import (
	"context"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectTracker interface {
	Add(obj client.Object)
	DeleteAll()
}

type ObjectTrackerBroker interface {
	ObjectTracker
	Scoped(collection string) ObjectTracker
}

type DefaultObjectTrackerBroker struct {
	mu                *sync.Mutex
	factoryF          func() ObjectTracker
	defaultObjTracker ObjectTracker
	collections       map[string]ObjectTracker
}

func NewDefaultObjectTrackerBroker(factoryF func() ObjectTracker) ObjectTrackerBroker {
	return &DefaultObjectTrackerBroker{
		factoryF:          factoryF,
		defaultObjTracker: factoryF(),
		collections:       map[string]ObjectTracker{},
		mu:                &sync.Mutex{},
	}
}

func (b *DefaultObjectTrackerBroker) Scoped(collection string) ObjectTracker {
	b.mu.Lock()
	defer b.mu.Unlock()
	if tracker, ok := b.collections[collection]; ok {
		return tracker
	}
	b.collections[collection] = b.factoryF()
	return b.collections[collection]
}

func (b *DefaultObjectTrackerBroker) Add(obj client.Object) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.defaultObjTracker.Add(obj)
}

func (b *DefaultObjectTrackerBroker) DeleteAll() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.defaultObjTracker.DeleteAll()
	for _, tracker := range b.collections {
		tracker.DeleteAll()
	}
}

type DefaultObjectTracker struct {
	mu        sync.Mutex
	arr       []client.Object
	ctx       context.Context
	k8sClient client.Client
}

type NoopObjectTracker struct{}

func (n *NoopObjectTracker) Add(_ client.Object) {
}

func (n *NoopObjectTracker) DeleteAll() {
}

func NewNoopObjectTracker() ObjectTracker {
	return &NoopObjectTracker{}
}

func NewObjectTracker(ctx context.Context, k8sClient client.Client) ObjectTracker {
	return &DefaultObjectTracker{
		ctx:       ctx,
		arr:       []client.Object{},
		mu:        sync.Mutex{},
		k8sClient: k8sClient,
	}
}

func (o *DefaultObjectTracker) Add(obj client.Object) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.arr = append(o.arr, obj)
}

func (o *DefaultObjectTracker) DeleteAll() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, obj := range o.arr {
		// ensures no race conditions between context being cancelled in specific test fixtures
		_ = o.k8sClient.Delete(context.Background(), obj)
	}
}
