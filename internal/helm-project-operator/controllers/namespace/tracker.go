package namespace

import (
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
)

// Getter gets a namespace that has been stored in a register
type Getter interface {
	// Has implies that the namespace has been registered
	Has(name string) bool

	// Get retrieves a registered namespace
	Get(name string) (*corev1.Namespace, bool)

	// List returns the names all registered namespaces
	List() []string

	// Compare checks if all names from a given list match namespaces registered in this Tracker
	Compare([]string) error
}

// Tracker can store namespace references and get them
type Tracker interface {
	Getter

	// Set registers a namespace
	Set(namespace *corev1.Namespace)

	// Delete unregisters a namespace
	Delete(namespace *corev1.Namespace)

	List() []string
}

// NewTracker returns a new tracker that can track and get namespaces
func NewTracker() Tracker {
	return &namespaceTracker{
		namespaceMap: make(map[string]*corev1.Namespace),
	}
}

type namespaceTracker struct {
	namespaceMap map[string]*corev1.Namespace
	mapLock      sync.RWMutex
}

// Has implies that the namespace has been registered
func (r *namespaceTracker) Has(name string) bool {
	r.mapLock.RLock()
	defer r.mapLock.RUnlock()
	_, exists := r.namespaceMap[name]
	return exists
}

// Get retrieves a registered namespace
func (r *namespaceTracker) Get(name string) (*corev1.Namespace, bool) {
	r.mapLock.RLock()
	defer r.mapLock.RUnlock()
	ns, exists := r.namespaceMap[name]
	if !exists {
		return nil, false
	}
	return ns, true
}

// List returns the names all registered namespaces
func (r *namespaceTracker) List() []string {
	r.mapLock.RLock()
	defer r.mapLock.RUnlock()

	var namespaces []string
	for _, ns := range r.namespaceMap {
		namespaces = append(namespaces, ns.Name)
	}

	return namespaces
}

// Compare checks if all names from a given list match namespaces registered in this Tracker
func (r *namespaceTracker) Compare(names []string) error {
	r.mapLock.RLock()
	defer r.mapLock.RUnlock()

	for _, ns := range names {
		_, exists := r.namespaceMap[ns]
		if !exists {
			return fmt.Errorf("namespace %s has not been registered in namespace Tracker", ns)
		}
	}

	if len(names) != len(r.namespaceMap) {
		return fmt.Errorf("namespace Tracker contains namespaces not present in given list")
	}

	return nil
}

// Set registers a namespace
func (r *namespaceTracker) Set(namespace *corev1.Namespace) {
	r.mapLock.Lock()
	defer r.mapLock.Unlock()
	r.namespaceMap[namespace.Name] = namespace
}

// Delete unregisters a namespace
func (r *namespaceTracker) Delete(namespace *corev1.Namespace) {
	r.mapLock.Lock()
	defer r.mapLock.Unlock()
	delete(r.namespaceMap, namespace.Name)
}
