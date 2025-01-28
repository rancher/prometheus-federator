package objectset

import (
	"github.com/rancher/wrangler/pkg/relatedresource"
)

// keyFunc is a utility function that returns a relatedresource.Key from a namespace and a name
func keyFunc(namespace, name string) relatedresource.Key {
	return relatedresource.Key{
		Namespace: namespace,
		Name:      name,
	}
}
