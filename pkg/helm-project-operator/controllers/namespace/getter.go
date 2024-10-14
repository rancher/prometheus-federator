package namespace

import (
	"fmt"
	"sort"

	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"

	corecontroller "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ProjectGetter allows you to get target namespaces based on a project and identify namespaces as special namespaces in a project
type ProjectGetter interface {
	// IsProjectRegistrationNamespace returns whether to watch for ProjectHelmCharts in the provided namespace
	IsProjectRegistrationNamespace(namespace *corev1.Namespace) bool

	// IsSystemNamespace returns whether the provided namespace is considered a system namespace
	IsSystemNamespace(namespace *corev1.Namespace) bool

	// GetTargetProjectNamespaces returns the list of namespaces that should be targeted for a given ProjectHelmChart
	// Any namespace returned by this should not be a project registration namespace or a system namespace
	GetTargetProjectNamespaces(projectHelmChart *v1alpha1.ProjectHelmChart) ([]string, error)
}

// Checker is a function that checks a namespace object and returns true or false
type Checker func(namespace *corev1.Namespace) bool

// NewLabelBasedProjectGetter returns a ProjectGetter that gets target project namespaces that meet the following criteria:
// 1) Must have the same projectLabel value as the namespace where the ProjectHelmChart lives in
// 2) Must not be a project registration namespace
// 3) Must not be a system namespace
func NewLabelBasedProjectGetter(
	projectLabel string,
	isProjectRegistrationNamespace Checker,
	isSystemNamespace Checker,
	namespaces corecontroller.NamespaceController,
) ProjectGetter {
	return &projectGetter{
		namespaces: namespaces,

		isProjectRegistrationNamespace: isProjectRegistrationNamespace,
		isSystemNamespace:              isSystemNamespace,

		getProjectNamespaces: func(projectHelmChart *v1alpha1.ProjectHelmChart) (*corev1.NamespaceList, error) {
			// source of truth is the projectLabel pair that exists on the namespace that the ProjectHelmChart lives within
			namespace, err := namespaces.Get(projectHelmChart.Namespace, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					// The projectHelmChart is not in a namespace that exists anymore, this implies it may have been deleted
					// Therefore, there are no project namespaces associated with this ProjectHelmChart
					return nil, nil
				}
				return nil, err
			}
			projectLabelValue, ok := namespace.Labels[projectLabel]
			if !ok {
				return nil, fmt.Errorf("could not find value of label %s in namespace %s", projectLabel, namespace.Name)
			}
			return namespaces.List(metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", projectLabel, projectLabelValue),
			})
		},
	}
}

// NewSingleNamespaceProjectGetter returns a ProjectGetter that gets target project namespaces that meet the following criteria:
// 1) Must match the labels provided on spec.projectNamespaceSelector of the projectHelmChart in question
// 2) Must not be the registration namespace
// 3) Must not be part of the provided systemNamespaces
func NewSingleNamespaceProjectGetter(
	registrationNamespace string,
	systemNamespaces []string,
	namespaces corecontroller.NamespaceController,
) ProjectGetter {
	isSystemNamespace := make(map[string]bool)
	for _, ns := range systemNamespaces {
		isSystemNamespace[ns] = true
	}
	return &projectGetter{
		namespaces: namespaces,

		isProjectRegistrationNamespace: func(namespace *corev1.Namespace) bool {
			// only one registrationNamespace exists
			return namespace.Name == registrationNamespace
		},
		isSystemNamespace: func(namespace *corev1.Namespace) bool {
			// only track explicit systemNamespaces
			return isSystemNamespace[namespace.Name]
		},

		getProjectNamespaces: func(projectHelmChart *v1alpha1.ProjectHelmChart) (*corev1.NamespaceList, error) {
			// source of truth is the ProjectHelmChart spec.projectNamespaceSelector
			selector, err := metav1.LabelSelectorAsSelector(projectHelmChart.Spec.ProjectNamespaceSelector)
			if err != nil {
				return nil, err
			}
			// List does not support the ability to ask for specific namespaces
			// based on a metav1.LabelSelector, so we get everything and then filter
			namespaceList, err := namespaces.List(metav1.ListOptions{})
			if err != nil {
				return nil, nil
			}
			if namespaceList == nil {
				return nil, nil
			}
			var namespaces []corev1.Namespace
			for _, ns := range namespaceList.Items {
				if !selector.Matches(labels.Set(ns.Labels)) {
					continue
				}
				namespaces = append(namespaces, ns)
			}
			namespaceList.Items = namespaces
			return namespaceList, nil
		},
	}
}

type projectGetter struct {
	namespaces corecontroller.NamespaceController

	isProjectRegistrationNamespace Checker
	isSystemNamespace              Checker

	getProjectNamespaces func(projectHelmChart *v1alpha1.ProjectHelmChart) (*corev1.NamespaceList, error)
}

// IsProjectRegistrationNamespace returns whether to watch for ProjectHelmCharts in the provided namespace
func (g *projectGetter) IsProjectRegistrationNamespace(namespace *corev1.Namespace) bool {
	return g.isProjectRegistrationNamespace(namespace)
}

// IsSystemNamespace returns whether the provided namespace is considered a system namespace
func (g *projectGetter) IsSystemNamespace(namespace *corev1.Namespace) bool {
	return g.isSystemNamespace(namespace)
}

// GetTargetProjectNamespaces returns the list of namespaces that should be targeted for a given ProjectHelmChart
// Any namespace returned by this should not be a project registration namespace or a system namespace
func (g *projectGetter) GetTargetProjectNamespaces(projectHelmChart *v1alpha1.ProjectHelmChart) ([]string, error) {
	projectNamespaceList, err := g.getProjectNamespaces(projectHelmChart)
	if err != nil {
		return nil, err
	}
	if projectNamespaceList == nil {
		return nil, nil
	}
	var namespaces []string
	for _, ns := range projectNamespaceList.Items {
		if g.isProjectRegistrationNamespace(&ns) || g.isSystemNamespace(&ns) {
			continue
		}
		namespaces = append(namespaces, ns.Name)
	}
	sort.Strings(namespaces)
	return namespaces, nil
}
