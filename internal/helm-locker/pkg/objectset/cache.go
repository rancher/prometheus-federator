package objectset

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rancher/helm-locker/pkg/gvk"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/objectset"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// LockableRegister implements Register and Locker
type LockableRegister interface {
	Register
	Locker
}

// Register can keep track of sets of ObjectSets
type Register interface {
	relatedresource.Enqueuer

	// Set allows you to set and lock an objectset associated with a specific key
	// if os or locked are not provided, the currently persisted values will be used
	Set(key relatedresource.Key, os *objectset.ObjectSet, locked *bool)

	// Delete allows you to delete an objectset associated with a specific key
	Delete(key relatedresource.Key, purge bool)
}

// Locker can lock or unlock object sets tied to a specific key
type Locker interface {
	// Lock allows you to lock an objectset associated with a specific key
	Lock(key relatedresource.Key)

	// Unlock allows you to unlock an objectset associated with a specific key
	Unlock(key relatedresource.Key)
}

// newLockableObjectSetRegisterAndCache returns:
// 1) a LockableRegister that allows registering new ObjectSets, locking them, unlocking them, or deleting them
// 2) a cache.SharedIndexInformer that listens to events on objectSetStates that are created from interacting with the provided register
//
// Note: This function is intentionally internal since the cache.SharedIndexInformer responds to an internal runtime.Object type (objectSetState)
func newLockableObjectSetRegisterAndCache(scf controller.SharedControllerFactory, triggerOnDelete func(string, bool)) (LockableRegister, cache.SharedIndexInformer) {
	c := lockableObjectSetRegisterAndCache{
		stateByKey:            make(map[relatedresource.Key]*objectSetState),
		keyByResourceKeyByGVK: make(map[schema.GroupVersionKind]map[relatedresource.Key]relatedresource.Key),

		stateChanges: make(chan watch.Event, 50),

		triggerOnDelete: triggerOnDelete,
	}
	// initialize watcher that populates watch queue
	c.gvkWatcher = gvk.NewWatcher(scf, c.Resolve, &c)
	// initialize informer
	c.SharedIndexInformer = cache.NewSharedIndexInformer(&c, &objectSetState{}, 10*time.Hour, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})
	return &c, &c
}

// lockableObjectSetRegisterAndCache is a cache.SharedIndexInformer that operates on objectSetStates
// and implements the LockableRegister interface via the informer
//
// internal note: also implements cache.ListerWatcher on objectSetStates
// internal note: also implements watch.Interface on objectSetStates
type lockableObjectSetRegisterAndCache struct {
	cache.SharedIndexInformer

	// stateChanges is the internal channel tracking events that happen to ObjectSetStates
	stateChanges chan watch.Event
	// gvkWatcher watches all GVKs tied to resources tracked by any ObjectSet tracked by this register
	// It will automatically trigger an Enqueue on seeing changes, which will trigger an event that
	// the underlying cache.SharedIndexInformer will process
	gvkWatcher gvk.Watcher
	// started represents whether the cache has been started yet
	started bool
	// startLock is a lock that prevents a Watch from occurring before the Informer has been started
	startLock sync.RWMutex

	// stateByKey is a map that keeps track of the desired state of the Register
	stateByKey map[relatedresource.Key]*objectSetState
	// stateMapLock is a lock on the stateByKey map
	stateMapLock sync.RWMutex

	// keyByResourceKeyByGVK is a map that keeps track of which resources are tied to a particular ObjectSet
	// This is used to make resolving the objectset on seeing changes to underlying resources more efficient
	keyByResourceKeyByGVK map[schema.GroupVersionKind]map[relatedresource.Key]relatedresource.Key
	// keyMapLock is a lock on the keyByResourceKeyByGVK map
	keyMapLock sync.RWMutex

	// triggerOnDelete allows registering a function that gets called on a delete from the cache
	// purge indicates whether or not the triggerOnDelete function is expected to purge underlying
	// resources on deleting an objectSet
	triggerOnDelete func(key string, purge bool)
}

// init initializes the register and the cache
func (c *lockableObjectSetRegisterAndCache) init() {
	c.startLock.Lock()
	defer c.startLock.Unlock()
	// do not start twice
	if !c.started {
		c.started = true
	}
}

// Run starts the objectSetState informer and starts watching GVKs tracked by ObjectSets
func (c *lockableObjectSetRegisterAndCache) Run(stopCh <-chan struct{}) {
	c.init()
	err := c.gvkWatcher.Start(context.TODO(), 50)
	if err != nil {
		logrus.Errorf("unable to watch gvks: %s", err)
	}

	c.SharedIndexInformer.Run(stopCh)
}

// Stop is a noop
// Allows implementing watch.Interface on objectSetStates
func (c *lockableObjectSetRegisterAndCache) Stop() {}

// ResultChan returns the channel that watch.Events on objectSetStates are registered on
// Allows implementing watch.Interface on objectSetStates
func (c *lockableObjectSetRegisterAndCache) ResultChan() <-chan watch.Event {
	return c.stateChanges
}

// List returns an objectSetStateList
// Allows implementing cache.ListerWatcher on objectSetStates
func (c *lockableObjectSetRegisterAndCache) List(options metav1.ListOptions) (runtime.Object, error) {
	c.stateMapLock.RLock()
	defer c.stateMapLock.RUnlock()
	objectSetStateList := &objectSetStateList{}
	for _, objectSetState := range c.stateByKey {
		if objectSetState != nil {
			objectSetStateList.Items = append(objectSetStateList.Items, *objectSetState)
		}
	}
	objectSetStateList.ResourceVersion = options.ResourceVersion
	return objectSetStateList, nil
}

// List returns an watch.Interface if the cache has been started that watches for events on objectSetStates
// Allows implementing cache.ListerWatcher on objectSetStates
func (c *lockableObjectSetRegisterAndCache) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	c.startLock.RLock()
	defer c.startLock.RUnlock()
	if !c.started {
		return nil, fmt.Errorf("cache is not started yet")
	}
	return c, nil
}

// Set allows you to set and lock an objectset associated with a specific key
func (c *lockableObjectSetRegisterAndCache) Set(key relatedresource.Key, os *objectset.ObjectSet, locked *bool) {
	logrus.Debugf("set objectset for %s/%s", key.Namespace, key.Name)
	c.setState(key, os, locked, false)
}

// Lock allows you to lock an objectset associated with a specific key
func (c *lockableObjectSetRegisterAndCache) Lock(key relatedresource.Key) {
	logrus.Debugf("locking %s/%s", key.Namespace, key.Name)
	s, ok := c.getState(key)
	if !ok {
		// nothing to lock
		return
	}
	s.mutateMu.RLock()
	defer s.mutateMu.RUnlock()
	if s.ObjectSet == nil {
		// nothing to lock
		return
	}
	c.lock(key, s.ObjectSet)
}

// Unlock allows you to unlock an objectset associated with a specific key
func (c *lockableObjectSetRegisterAndCache) Unlock(key relatedresource.Key) {
	logrus.Debugf("unlocking %s/%s", key.Namespace, key.Name)
	c.unlock(key)
}

// Delete allows you to delete an objectset associated with a specific key
func (c *lockableObjectSetRegisterAndCache) Delete(key relatedresource.Key, purge bool) {
	logrus.Debugf("deleting %s/%s", key.Namespace, key.Name)
	c.deleteState(key)
	c.triggerOnDelete(fmt.Sprintf("%s/%s", key.Namespace, key.Name), purge)
}

// Enqueue allows you to enqueue an objectset associated with a specific key
func (c *lockableObjectSetRegisterAndCache) Enqueue(namespace, name string) {
	key := keyFunc(namespace, name)
	c.setState(key, nil, nil, true)
}

// Resolve allows you to resolve an object seen in the cluster to an ObjectSet tracked in this LockableRegister
// Objects will only be resolved if the LockableRegister has locked this ObjectSet
func (c *lockableObjectSetRegisterAndCache) Resolve(gvk schema.GroupVersionKind, namespace, name string, _ runtime.Object) ([]relatedresource.Key, error) {
	resourceKey := keyFunc(namespace, name)

	c.keyMapLock.RLock()
	defer c.keyMapLock.RUnlock()
	keyByResourceKey, ok := c.keyByResourceKeyByGVK[gvk]
	if !ok {
		// do nothing since we're not watching this GVK anymore
		return nil, nil
	}
	key, ok := keyByResourceKey[resourceKey]
	if !ok {
		// do nothing since the resource is not tied to a set
		return nil, nil
	}
	logrus.Infof("detected change in %s/%s (%s), enqueuing objectset %s/%s", namespace, name, gvk, key.Namespace, key.Name)
	return []relatedresource.Key{key}, nil
}

// getState returns the underlying objectSetState for a given key
func (c *lockableObjectSetRegisterAndCache) getState(key relatedresource.Key) (*objectSetState, bool) {
	c.stateMapLock.RLock()
	defer c.stateMapLock.RUnlock()
	state, ok := c.stateByKey[key]
	return state, ok
}

// setState allows a user to set the objectSetState for a given key
func (c *lockableObjectSetRegisterAndCache) setState(key relatedresource.Key, os *objectset.ObjectSet, locked *bool, forceEnqueue bool) {
	// get old state and use as the base
	originalState, modifying := c.getState(key)
	var s *objectSetState

	// generate new state to be set
	if !modifying {
		s = newObjectSetState(key.Namespace, key.Name, objectSetState{})
	} else {
		s = originalState.DeepCopy()
		s.Generation++
		s.ResourceVersion = fmt.Sprintf("%d", s.Generation)
	}

	// apply provided settings or use original state as default
	if os != nil {
		s.ObjectSet = os
	}
	if locked != nil {
		s.Locked = *locked
	}

	// do nothing if the object has not changed
	objectChanged := forceEnqueue || !modifying
	if modifying {
		objectChanged = objectChanged || s.ObjectSet != originalState.ObjectSet || s.Locked != originalState.Locked
	}
	if !objectChanged {
		return
	}

	// handle adding events and storing state
	c.stateMapLock.Lock()
	defer c.stateMapLock.Unlock()
	if modifying {
		c.stateChanges <- watch.Event{Type: watch.Modified, Object: s}
	} else {
		c.stateChanges <- watch.Event{Type: watch.Added, Object: s}
	}
	c.stateByKey[key] = s
	logrus.Debugf("set state for %s/%s: locked %t, os %p, objectMeta: %v", s.Namespace, s.Name, s.Locked, s.ObjectSet, s.ObjectMeta)
}

// deleteState deletes anything on the register for a given key
func (c *lockableObjectSetRegisterAndCache) deleteState(key relatedresource.Key) {
	s, exists := c.getState(key)
	if !exists {
		// nothing to add, event was already processed
		return
	}
	c.stateMapLock.Lock()
	delete(c.stateByKey, key)
	c.stateMapLock.Unlock()
	s.mutateMu.Lock()
	s.ObjectSet = nil
	s.mutateMu.Unlock()
	s.Locked = false
	c.stateChanges <- watch.Event{Type: watch.Deleted, Object: s}
}

// lock adds entries to the register to ensure that resources tracked by this ObjectSet are resolved to this ObjectSet
func (c *lockableObjectSetRegisterAndCache) lock(key relatedresource.Key, os *objectset.ObjectSet) error {
	c.keyMapLock.Lock()
	defer c.keyMapLock.Unlock()

	if err := c.canLock(key, os); err != nil {
		return err
	}

	c.removeAllEntries(key)

	objectsByGVK := os.ObjectsByGVK()

	for gvk, objMap := range objectsByGVK {
		keyByResourceKey, ok := c.keyByResourceKeyByGVK[gvk]
		if !ok {
			keyByResourceKey = make(map[relatedresource.Key]relatedresource.Key)
		}
		for objKey := range objMap {
			resourceKey := keyFunc(objKey.Namespace, objKey.Name)
			keyByResourceKey[resourceKey] = key
		}
		c.keyByResourceKeyByGVK[gvk] = keyByResourceKey

		// ensure that we are watching this new GVK
		if err := c.gvkWatcher.Watch(gvk); err != nil {
			return err
		}
	}

	return nil
}

// unlock removes all entries to the register tied to a particular ObjectSet by key
func (c *lockableObjectSetRegisterAndCache) unlock(key relatedresource.Key) {
	c.keyMapLock.Lock()
	defer c.keyMapLock.Unlock()

	c.removeAllEntries(key)
}

// canLock returns whether trynig to lock the provided ObjectSet will result in an error
// One of the few reasons why this is possible is if two registered ObjectSets are attempting to track the same resource
func (c *lockableObjectSetRegisterAndCache) canLock(key relatedresource.Key, os *objectset.ObjectSet) error {
	objectsByGVK := os.ObjectsByGVK()
	for gvk, objMap := range objectsByGVK {
		keyByResourceKey, ok := c.keyByResourceKeyByGVK[gvk]
		if !ok {
			continue
		}
		for objKey := range objMap {
			resourceKey := keyFunc(objKey.Namespace, objKey.Name)
			currKey, ok := keyByResourceKey[resourceKey]
			if ok && currKey != key {
				// object is already associated with another set
				return fmt.Errorf("cannot lock objectset for %s: object %s is already associated with key %s", key, objKey, currKey)
			}
		}
	}
	return nil
}

// removeAllEntries removes all entries to the register tied to a particular ObjectSet by key
// Note: This is a thread-unsafe version of
func (c *lockableObjectSetRegisterAndCache) removeAllEntries(key relatedresource.Key) {
	for gvk, keyByResourceKey := range c.keyByResourceKeyByGVK {
		for resourceKey, currSetKey := range keyByResourceKey {
			if key == currSetKey {
				delete(keyByResourceKey, resourceKey)
			}
		}
		if len(keyByResourceKey) == 0 {
			delete(c.keyByResourceKeyByGVK, gvk)
		} else {
			c.keyByResourceKeyByGVK[gvk] = keyByResourceKey
		}
	}
}
