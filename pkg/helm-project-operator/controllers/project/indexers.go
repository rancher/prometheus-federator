package project

import (
	"fmt"

	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	common2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// All namespaces
const (
	// ProjectHelmChartByReleaseName identifies a ProjectHelmChart by the underlying Helm release it is tied to
	ProjectHelmChartByReleaseName = "helm.cattle.io/project-helm-chart-by-release-name"
)

// Registration namespaces only
const (
	// RoleBindingInRegistrationNamespaceByRoleRef identifies the set of RoleBindings in a registration namespace
	// that are tied to specific RoleRefs that need to be watched by the operator
	RoleBindingInRegistrationNamespaceByRoleRef = "helm.cattle.io/role-binding-in-registration-ns-by-role-ref"

	// ClusterRoleBindingByRoleRef identifies the set of ClusterRoleBindings that are tied to RoleRefs that need
	// to be watched by the operator
	ClusterRoleBindingByRoleRef = "helm.cattle.io/cluster-role-binding-by-role-ref"

	// BindingReferencesDefaultOperatorRole is the value of the both of the above indices when a ClusterRoleBinding or RoleBinding
	// is tied to a RoleRef that matches a default ClusterRole that is watched by the operator to create admin, edit, or view RoleBindings
	// in the Project Release Namespace
	BindingReferencesDefaultOperatorRole = "bound-to-default-role"
)

// NamespacedBindingReferencesDefaultOperatorRole is the index used to mark a RoleBinding as one that targets
// one of the default operator roles (supplied in RuntimeOptions under AdminClusterRole, EditClusterRole, and ViewClusterRole)
func NamespacedBindingReferencesDefaultOperatorRole(namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, BindingReferencesDefaultOperatorRole)
}

// Release namespaces only
const (
	// RoleInReleaseNamespaceByReleaseNamespaceName identifies a Role in a release namespace that needs to have RBAC synced
	// on changes to RoleBindings in the Project Registration Namespace or ClusterRoleBindings.
	// The value of this will be the namespace and name of the Helm release that it is for.
	RoleInReleaseNamespaceByReleaseNamespaceName = "helm.cattle.io/role-in-release-ns-by-release-namespace-name"

	// ConfigMapInReleaseNamespaceByReleaseNamespaceName identifies a ConfigMap in a release namespace that is tied to the
	// ProjectHelmChart's status in the release namespace.
	// The value of this will be the namespace and name of the Helm release that it is for.
	ConfigMapInReleaseNamespaceByReleaseNamespaceName = "helm.cattle.io/configmap-in-release-ns-by-release-namespace-name"
)

// initIndexers initializes indexers that allow for more efficient computations on related resources without relying on additional
// calls to be made to the Kubernetes API by referencing the cache instead
func (h *handler) initIndexers() {
	h.projectHelmChartCache.AddIndexer(ProjectHelmChartByReleaseName, h.projectHelmChartToReleaseName)

	h.rolebindingCache.AddIndexer(RoleBindingInRegistrationNamespaceByRoleRef, h.roleBindingInRegistrationNamespaceToRoleRef)

	h.clusterrolebindingCache.AddIndexer(ClusterRoleBindingByRoleRef, h.clusterRoleBindingToRoleRef)

	h.roleCache.AddIndexer(RoleInReleaseNamespaceByReleaseNamespaceName, h.roleInReleaseNamespaceToReleaseNamespaceName)

	h.configmapCache.AddIndexer(ConfigMapInReleaseNamespaceByReleaseNamespaceName, h.configMapInReleaseNamespaceToReleaseNamespaceName)
}

func (h *handler) projectHelmChartToReleaseName(projectHelmChart *v1alpha1.ProjectHelmChart) ([]string, error) {
	shouldManage := h.shouldManage(projectHelmChart)
	if !shouldManage {
		return nil, nil
	}
	_, releaseName := h.getReleaseNamespaceAndName(projectHelmChart)
	return []string{releaseName}, nil
}

func (h *handler) roleBindingInRegistrationNamespaceToRoleRef(rb *rbacv1.RoleBinding) ([]string, error) {
	if rb == nil {
		return nil, nil
	}
	namespace, err := h.namespaceCache.Get(rb.Namespace)
	if err != nil {
		// If we can't get the namespace the rolebinding resides in role binding resides in does not exist, we don't need to index
		// it since it's probably gotten deleted anyways.
		//
		// Note: we know that this error would only happen if the namespace is not found since the only valid error returned from this
		// call is errors.NewNotFound(c.resource, name)
		return nil, nil
	}
	isProjectRegistrationNamespace := h.projectGetter.IsProjectRegistrationNamespace(namespace)
	if !isProjectRegistrationNamespace {
		return nil, nil
	}
	_, isDefaultRoleRef := common2.IsDefaultClusterRoleRef(h.opts, rb.RoleRef.Name)
	if !isDefaultRoleRef {
		// we only care about rolebindings in the registration namespace that are tied to the default roles
		// created by this operator
		return nil, nil
	}
	// keep track of this rolebinding in the index so we can grab it later
	return []string{NamespacedBindingReferencesDefaultOperatorRole(rb.Namespace)}, nil
}

func (h *handler) clusterRoleBindingToRoleRef(crb *rbacv1.ClusterRoleBinding) ([]string, error) {
	if crb == nil {
		return nil, nil
	}
	_, isDefaultRoleRef := common2.IsDefaultClusterRoleRef(h.opts, crb.RoleRef.Name)
	if !isDefaultRoleRef {
		// we only care about rolebindings in the registration namespace that are tied to the default roles
		// created by this operator
		return nil, nil
	}
	// keep track of this rolebinding in the index so we can grab it later
	return []string{BindingReferencesDefaultOperatorRole}, nil
}

func (h *handler) roleInReleaseNamespaceToReleaseNamespaceName(role *rbacv1.Role) ([]string, error) {
	if role == nil {
		return nil, nil
	}
	return h.getReleaseIndexFromNamespaceAndLabels(role.Namespace, role.Labels, common2.HelmProjectOperatorProjectHelmChartRoleLabel)
}

func (h *handler) configMapInReleaseNamespaceToReleaseNamespaceName(configmap *corev1.ConfigMap) ([]string, error) {
	if configmap == nil {
		return nil, nil
	}
	return h.getReleaseIndexFromNamespaceAndLabels(configmap.Namespace, configmap.Labels, common2.HelmProjectOperatorDashboardValuesConfigMapLabel)
}

func (h *handler) getReleaseIndexFromNamespaceAndLabels(namespace string, labels map[string]string, releaseLabel string) ([]string, error) {
	if labels == nil {
		return nil, nil
	}
	releaseName, ok := labels[releaseLabel]
	if !ok {
		return nil, nil
	}

	return []string{fmt.Sprintf("%s/%s", namespace, releaseName)}, nil
}
