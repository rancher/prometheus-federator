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
		_ = o.k8sClient.Delete(o.ctx, obj)
	}
}
