package project

import (
	helmcontrollerv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	"github.com/k3s-io/helm-controller/pkg/controllers/chart"
	helmlockerv1alpha1 "github.com/rancher/prometheus-federator/pkg/helm-locker/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/pkg/helm-locker/controllers/release"
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	common2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Note: each resource created here should have a resolver set in resolvers.go
// The only exception is ProjectHelmCharts since those are handled by the main generating controller

// getHelmChart returns the HelmChart created on behalf of this ProjectHelmChart
func (h *handler) getHelmChart(projectID string, valuesContent string, projectHelmChart *v1alpha1.ProjectHelmChart) *helmcontrollerv1.HelmChart {
	// must be in system namespace since helm controllers are configured to only watch one namespace
	jobImage := DefaultJobImage
	if len(h.opts.HelmJobImage) > 0 {
		jobImage = h.opts.HelmJobImage
	}
	releaseNamespace, releaseName := h.getReleaseNamespaceAndName(projectHelmChart)
	helmChart := helmcontrollerv1.NewHelmChart(h.systemNamespace, releaseName, helmcontrollerv1.HelmChart{
		Spec: helmcontrollerv1.HelmChartSpec{
			TargetNamespace: releaseNamespace,
			Chart:           releaseName,
			JobImage:        jobImage,
			ChartContent:    h.opts.ChartContent,
			ValuesContent:   valuesContent,
		},
	})
	helmChart.SetLabels(common2.GetHelmResourceLabels(projectID, projectHelmChart.Spec.HelmAPIVersion))
	helmChart.SetAnnotations(map[string]string{
		chart.ManagedBy: h.opts.ControllerName,
	})
	return helmChart
}

// getHelmRelease returns the HelmRelease created on behalf of this ProjectHelmChart
func (h *handler) getHelmRelease(projectID string, projectHelmChart *v1alpha1.ProjectHelmChart) *helmlockerv1alpha1.HelmRelease {
	// must be in system namespace since helmlocker controllers are configured to only watch one namespace
	releaseNamespace, releaseName := h.getReleaseNamespaceAndName(projectHelmChart)
	helmRelease := helmlockerv1alpha1.NewHelmRelease(h.systemNamespace, releaseName, helmlockerv1alpha1.HelmRelease{
		Spec: helmlockerv1alpha1.HelmReleaseSpec{
			Release: helmlockerv1alpha1.ReleaseKey{
				Namespace: releaseNamespace,
				Name:      releaseName,
			},
		},
	})
	helmRelease.SetLabels(common2.GetHelmResourceLabels(projectID, projectHelmChart.Spec.HelmAPIVersion))
	helmRelease.SetAnnotations(map[string]string{
		release.ManagedBy: h.opts.ControllerName,
	})
	return helmRelease
}

// getProjectReleaseNamespace returns the Project Release Namespace created on behalf of this ProjectHelmChart, if required
func (h *handler) getProjectReleaseNamespace(projectID string, isOrphaned bool, projectHelmChart *v1alpha1.ProjectHelmChart) *v1.Namespace {
	releaseNamespace, _ := h.getReleaseNamespaceAndName(projectHelmChart)
	if releaseNamespace == h.systemNamespace || releaseNamespace == projectHelmChart.Namespace {
		return nil
	}
	projectReleaseNamespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        releaseNamespace,
			Annotations: common2.GetProjectNamespaceAnnotations(h.opts.ProjectReleaseLabelValue, h.opts.ProjectLabel, h.opts.ClusterID),
			Labels:      common2.GetProjectNamespaceLabels(projectID, h.opts.ProjectLabel, h.opts.ProjectReleaseLabelValue, isOrphaned),
		},
	}
	return projectReleaseNamespace
}

// getRoleBindings returns the RoleBindings created on behalf of this ProjectHelmChart in the Project Release Namespace based on Roles created in the
// Project Release Namespace and RoleBindings attached to the default operator roles (configured as AdminClusterRole, EditClusterRole, and ViewClusterRole
//
//	in the providedRuntimeOptions) in the Project Registration Namespace only. To update these RoleBindings in the release namespace, you will need to assign
//
// additional permissions to the default roles in the Project Registration Namespace or manually assign RoleBindings in the release namespace.
func (h *handler) getRoleBindings(projectID string, k8sRoleToRoleRefs map[string][]rbacv1.RoleRef, k8sRoleToSubjects map[string][]rbacv1.Subject, projectHelmChart *v1alpha1.ProjectHelmChart) []runtime.Object {
	var objs []runtime.Object
	releaseNamespace, _ := h.getReleaseNamespaceAndName(projectHelmChart)

	for subjectRole := range common2.GetDefaultClusterRoles(h.opts) {
		// note: these role refs point to roles in the release namespace
		roleRefs := k8sRoleToRoleRefs[subjectRole]
		// note: these subjects are inferred from the rolebindings tied to the default roles in the registration namespace
		subjects := k8sRoleToSubjects[subjectRole]
		if len(subjects) == 0 {
			// no need to create empty RoleBindings
			continue
		}
		for _, roleRef := range roleRefs {
			objs = append(objs, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleRef.Name,
					Namespace: releaseNamespace,
					Labels:    common2.GetCommonLabels(projectID),
				},
				RoleRef:  roleRef,
				Subjects: subjects,
			})
		}
	}

	return objs
}
