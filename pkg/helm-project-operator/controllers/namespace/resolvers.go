package namespace

import (
	"context"

	"github.com/rancher/wrangler/pkg/relatedresource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Note: each resource created in resources.go should have a resolver handler here
// The only exception is namespaces since those are handled by the main controller OnChange

// initResolvers initializes resolvers that need to be set to watch child resources of Project Registration Namespaces
func (h *handler) initResolvers(ctx context.Context) {
	relatedresource.WatchClusterScoped(
		ctx, "watch-project-registration-namespace-data", h.resolveProjectRegistrationNamespaceData, h.namespaces,
		h.configmaps,
	)
}

func (h *handler) resolveProjectRegistrationNamespaceData(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
	if !h.projectRegistrationNamespaceTracker.Has(namespace) {
		// no longer need to watch for changes to resources in this namespace since it is no longer tracked
		// if the namespace ever becomes unorphaned, we can track it again
		return nil, nil
	}
	if obj == nil {
		return nil, nil
	}
	if configmap, ok := obj.(*corev1.ConfigMap); ok {
		return h.resolveConfigMap(namespace, name, configmap)
	}
	return nil, nil
}

func (h *handler) resolveConfigMap(namespace, name string, _ *corev1.ConfigMap) ([]relatedresource.Key, error) {
	// check if name matches
	if name == h.getConfigMapName() {
		return []relatedresource.Key{{
			Name: namespace,
		}}, nil
	}
	return nil, nil
}
