package project

import (
	"context"

	common2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"

	helmcontrollerv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	helmlockerv1alpha1 "github.com/rancher/prometheus-federator/pkg/helm-locker/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

// Note: each resource created in resources.go, registrationdata.go, or releasedata.go should have a resolver handler here
// The only exception is ProjectHelmCharts since those are handled by the main generating controller

// initResolvers initializes resolvers that need to be set to watch child resources of ProjectHelmCharts
func (h *handler) initResolvers(ctx context.Context) {
	if len(h.opts.ProjectLabel) != 0 && len(h.opts.ProjectReleaseLabelValue) == 0 {
		// Only trigger watching project release namespace if it is created by the operator
		relatedresource.Watch(
			ctx, "watch-project-release-namespace", h.resolveProjectReleaseNamespace, h.projectHelmCharts,
			h.namespaces,
		)
	}

	relatedresource.Watch(
		ctx, "watch-system-namespace-chart-data", h.resolveSystemNamespaceData, h.projectHelmCharts,
		h.helmCharts, h.helmReleases,
	)

	relatedresource.Watch(
		ctx, "watch-project-registration-chart-data", h.resolveProjectRegistrationNamespaceData, h.projectHelmCharts,
		h.rolebindings, h.clusterrolebindings,
	)

	relatedresource.Watch(
		ctx, "watch-project-release-chart-data", h.resolveProjectReleaseNamespaceData, h.projectHelmCharts,
		h.rolebindings, h.configmaps, h.roles,
	)
}

// Project Release Namespace

func (h *handler) resolveProjectReleaseNamespace(_, _ string, obj runtime.Object) ([]relatedresource.Key, error) {
	if obj == nil {
		return nil, nil
	}
	ns, ok := obj.(*corev1.Namespace)
	if !ok {
		return nil, nil
	}
	// since the release namespace will be created and owned by the ProjectHelmChart,
	// we can simply leverage is annotations to identify what we should resolve to.
	// If the release namespace is orphaned, the owner annotation should be removed automatically
	return h.resolveProjectHelmChartOwned(ns.Annotations)
}

// System Namespace Data

func (h *handler) resolveSystemNamespaceData(namespace, _ string, obj runtime.Object) ([]relatedresource.Key, error) {
	if namespace != h.systemNamespace {
		return nil, nil
	}
	if obj == nil {
		return nil, nil
	}
	// since the HelmChart and HelmRelease will be created and owned by the ProjectHelmChart,
	// we can simply leverage is annotations to identify what we should resolve to.
	if helmChart, ok := obj.(*helmcontrollerv1.HelmChart); ok {
		return h.resolveProjectHelmChartOwned(helmChart.Annotations)
	}
	if helmRelease, ok := obj.(*helmlockerv1alpha1.HelmRelease); ok {
		return h.resolveProjectHelmChartOwned(helmRelease.Annotations)
	}
	return nil, nil
}

// Project Registration Namespace Data

func (h *handler) resolveProjectRegistrationNamespaceData(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
	//h.projectHelmCharts, h.rolebindings, h.clusterrolebindings

	if obj == nil {
		return nil, nil
	}
	if rb, ok := obj.(*rbacv1.RoleBinding); ok {
		logrus.Debugf("Resolving project registration namespace rolebindings for %s", namespace)
		return h.resolveProjectRegistrationNamespaceRoleBinding(namespace, name, rb)
	}
	if crb, ok := obj.(*rbacv1.ClusterRoleBinding); ok {
		return h.resolveClusterRoleBinding(namespace, name, crb)
	}
	return nil, nil
}

func (h *handler) resolveProjectRegistrationNamespaceRoleBinding(namespace, _ string, rb *rbacv1.RoleBinding) ([]relatedresource.Key, error) {
	namespaceObj, err := h.namespaceCache.Get(namespace)
	if err != nil {
		logrus.Debugf("Namespace not found %s: ", namespace)
		return nil, err
	}
	isProjectRegistrationNamespace := h.projectGetter.IsProjectRegistrationNamespace(namespaceObj)
	if !isProjectRegistrationNamespace {
		logrus.Debugf("%s is not a project registration namespace: ", namespace)
		return nil, nil
	}

	// we want to re-enqueue the ProjectHelmChart if the rolebinding's ref points to one of the operator default roles
	_, isDefaultRoleRef := common2.IsDefaultClusterRoleRef(h.opts, rb.RoleRef.Name)
	if !isDefaultRoleRef {
		return nil, nil
	}
	// re-enqueue all HelmCharts in this project registration namespace
	projectHelmCharts, err := h.projectHelmChartCache.List(namespace, labels.Everything())
	if err != nil {
		logrus.Debugf("Error in resolveProjectRegistrationNamespaceRoleBinding while re-enqueuing HelmCharts in %s", namespace)
		return nil, err
	}
	var keys []relatedresource.Key
	for _, projectHelmChart := range projectHelmCharts {
		if projectHelmChart == nil {
			continue
		}
		keys = append(keys, relatedresource.Key{
			Namespace: namespace,
			Name:      projectHelmChart.Name,
		})
	}
	return keys, nil
}

func (h *handler) resolveClusterRoleBinding(_, _ string, crb *rbacv1.ClusterRoleBinding) ([]relatedresource.Key, error) {
	// we want to re-enqueue the ProjectHelmChart if the rolebinding's ref points to one of the operator default roles
	_, isDefaultRoleRef := common2.IsDefaultClusterRoleRef(h.opts, crb.RoleRef.Name)
	if !isDefaultRoleRef {
		return nil, nil
	}
	// re-enqueue all HelmCharts in all Project Registration namespaces
	namespaces, err := h.namespaceCache.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	var keys []relatedresource.Key
	for _, namespace := range namespaces {
		if namespace == nil {
			continue
		}
		isProjectRegistrationNamespace := h.projectGetter.IsProjectRegistrationNamespace(namespace)
		if !isProjectRegistrationNamespace {
			continue
		}
		projectHelmCharts, err := h.projectHelmChartCache.List(namespace.Name, labels.Everything())
		if err != nil {
			logrus.Debugf("Error in resolveClusterRoleBinding while re-enqueuing HelmCharts in %s", namespace)
			return nil, err
		}
		for _, projectHelmChart := range projectHelmCharts {
			if projectHelmChart == nil {
				continue
			}
			keys = append(keys, relatedresource.Key{
				Namespace: projectHelmChart.Namespace,
				Name:      projectHelmChart.Name,
			})
		}
	}
	return keys, nil
}

// Project Release Namespace Data

func (h *handler) resolveProjectReleaseNamespaceData(_, _ string, obj runtime.Object) ([]relatedresource.Key, error) {
	if obj == nil {
		return nil, nil
	}
	if rb, ok := obj.(*rbacv1.RoleBinding); ok {
		// since the rolebinding will be created and owned by the ProjectHelmChart,
		// we can simply leverage is annotations to identify what we should resolve to.
		return h.resolveProjectHelmChartOwned(rb.Annotations)
	}
	if configmap, ok := obj.(*corev1.ConfigMap); ok {
		return h.resolveByProjectReleaseLabelValue(configmap.Labels, common2.HelmProjectOperatorDashboardValuesConfigMapLabel)
	}
	if role, ok := obj.(*rbacv1.Role); ok {
		return h.resolveByProjectReleaseLabelValue(role.Labels, common2.HelmProjectOperatorProjectHelmChartRoleLabel)
	}
	return nil, nil
}

// Common

func (h *handler) resolveProjectHelmChartOwned(annotations map[string]string) ([]relatedresource.Key, error) {
	// Q: Why aren't we using relatedresource.OwnerResolver?
	// A: in k8s, you can't set an owner reference across namespaces, which means that when --project-label is provided
	// (where the ProjectHelmChart will be outside the systemNamespace where the HelmCharts and HelmReleases are created),
	// ownerReferences will not be set on the object. However, wrangler annotations will be set since those objects are
	// created via a wrangler apply. Therefore, we leverage those annotations to figure out which ProjectHelmChart to enqueue
	if annotations == nil {
		return nil, nil
	}
	ownerNamespace, ok := annotations[apply.LabelNamespace]
	if !ok {
		return nil, nil
	}
	ownerName, ok := annotations[apply.LabelName]
	if !ok {
		return nil, nil
	}

	return []relatedresource.Key{{
		Namespace: ownerNamespace,
		Name:      ownerName,
	}}, nil
}

func (h *handler) resolveByProjectReleaseLabelValue(labels map[string]string, projectReleaseLabel string) ([]relatedresource.Key, error) {
	if labels == nil {
		return nil, nil
	}
	releaseName, ok := labels[projectReleaseLabel]
	if !ok {
		return nil, nil
	}
	projectHelmCharts, err := h.projectHelmChartCache.GetByIndex(ProjectHelmChartByReleaseName, releaseName)
	if err != nil {
		return nil, err
	}
	var keys []relatedresource.Key
	for _, projectHelmChart := range projectHelmCharts {
		if projectHelmChart == nil {
			continue
		}
		keys = append(keys, relatedresource.Key{
			Namespace: projectHelmChart.Namespace,
			Name:      projectHelmChart.Name,
		})
	}
	return keys, nil
}
