package hardened

import (
	"context"

	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"

	"github.com/rancher/wrangler/pkg/relatedresource"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Note: each resource created in resources.go should have a resolver handler here
// The only exception is namespaces since those are handled by the main controller OnChange

// initResolvers initializes resolvers that need to be set to watch child resources of Helm Project Operated Namespaces
func (h *handler) initResolvers(ctx context.Context) {
	relatedresource.WatchClusterScoped(
		ctx, "watch-hardened-hpo-operated-namespace", h.resolveHardenedProjectRegistrationNamespaceData, h.namespaces,
		h.serviceaccounts, h.networkpolicies,
	)
}

func (h *handler) resolveHardenedProjectRegistrationNamespaceData(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
	if obj == nil {
		return nil, nil
	}
	ns, err := h.namespaceCache.Get(namespace)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		// namespace is probably being deleted, which means we don't need to resolve anything
		return nil, nil
	}
	if !common.HasHelmProjectOperatedLabel(ns.Labels) {
		// only care about service accounts and network policies in an operated namespace
		return nil, nil
	}
	if serviceAccount, ok := obj.(*corev1.ServiceAccount); ok {
		return h.resolveServiceAccount(namespace, name, serviceAccount)
	}
	if networkPolicy, ok := obj.(*networkingv1.NetworkPolicy); ok {
		return h.resolveNetworkPolicy(namespace, name, networkPolicy)
	}
	return nil, nil
}

func (h *handler) resolveServiceAccount(namespace, name string, _ *corev1.ServiceAccount) ([]relatedresource.Key, error) {
	// check if name matches
	if name == defaultServiceAccountName {
		return []relatedresource.Key{{
			Name: namespace,
		}}, nil
	}
	return nil, nil
}

func (h *handler) resolveNetworkPolicy(namespace, name string, _ *networkingv1.NetworkPolicy) ([]relatedresource.Key, error) {
	// check if name matches
	if name == defaultNetworkPolicyName {
		return []relatedresource.Key{{
			Name: namespace,
		}}, nil
	}
	return nil, nil
}
