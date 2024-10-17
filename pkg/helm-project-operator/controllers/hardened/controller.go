package hardened

import (
	"context"

	common2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"

	"github.com/rancher/wrangler/pkg/apply"
	corecontroller "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	networkingcontroller "github.com/rancher/wrangler/pkg/generated/controllers/networking.k8s.io/v1"
	corev1 "k8s.io/api/core/v1"
)

type handler struct {
	apply apply.Apply

	opts common2.HardeningOptions

	namespaces      corecontroller.NamespaceController
	namespaceCache  corecontroller.NamespaceCache
	serviceaccounts corecontroller.ServiceAccountController
	networkpolicies networkingcontroller.NetworkPolicyController
}

func Register(
	ctx context.Context,
	apply apply.Apply,
	_ common2.HardeningOptions,
	namespaces corecontroller.NamespaceController,
	namespaceCache corecontroller.NamespaceCache,
	serviceaccounts corecontroller.ServiceAccountController,
	networkpolicies networkingcontroller.NetworkPolicyController,
) {

	apply = apply.
		WithSetID("hardened-hpo-operated-namespace").
		WithCacheTypes(serviceaccounts, networkpolicies)

	h := &handler{
		apply:           apply,
		namespaces:      namespaces,
		namespaceCache:  namespaceCache,
		serviceaccounts: serviceaccounts,
		networkpolicies: networkpolicies,
	}

	h.initResolvers(ctx)

	namespaces.OnChange(ctx, "harden-hpo-operated-namespace", h.OnChange)
}

func (h *handler) OnChange(_ string, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	if namespace == nil {
		return namespace, nil
	}
	if namespace.DeletionTimestamp != nil {
		// When a namespace gets deleted, all resources deployed to harden that namespace should also get deleted
		// Therefore, we do not need to apply anything in this situation to avoid spamming logs with trying to apply
		// a resource to a namespace that is being terminated
		return namespace, nil
	}
	if !common2.HasHelmProjectOperatedLabel(namespace.Labels) {
		// only harden operated namespaces
		return namespace, nil
	}
	return namespace, h.apply.WithOwner(namespace).ApplyObjects(
		h.getDefaultServiceAccount(namespace),
		h.getNetworkPolicy(namespace),
	)
}
