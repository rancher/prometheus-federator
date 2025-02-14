package gvk

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// Lister is any object that can list a set of GVKs or return an error
type Lister interface {
	List() ([]schema.GroupVersionKind, error)
}

// NewLister returns an object that implements the Lister interface
func NewLister(discovery discovery.DiscoveryInterface) Lister {
	return &lister{
		discovery: discovery,
	}
}

// lister implements the Lister interface given the provided discovery interface
type lister struct {
	discovery discovery.DiscoveryInterface
}

// List returns a list of schema.GroupVersionKinds that you can run informers on
func (l *lister) List() ([]schema.GroupVersionKind, error) {
	_, resources, err := l.discovery.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}
	var gvks []schema.GroupVersionKind
	for _, resource := range resources {
		for _, apiResource := range resource.APIResources {
			if strings.Contains(apiResource.Name, "/") {
				// Ignore subresources
				continue
			}
			gvks = append(gvks, schema.FromAPIVersionAndKind(resource.GroupVersion, apiResource.Kind))
		}
	}
	return gvks, nil
}
