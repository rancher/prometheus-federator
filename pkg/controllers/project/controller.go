package project

import (
	"context"
	"fmt"

	monitoring "github.com/aiyengar2/prometheus-federator/pkg/apis/monitoring.cattle.io/v1alpha1"
	monitoringcontrollers "github.com/aiyengar2/prometheus-federator/pkg/generated/controllers/monitoring.cattle.io/v1alpha1"
	prometheusoperatorcontrollers "github.com/aiyengar2/prometheus-federator/pkg/generated/controllers/monitoring.coreos.com/v1"
	"github.com/rancher/wrangler/pkg/apply"
	corecontrollers "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

type handler struct {
	systemNamespace string

	clusterPrometheusName      string
	clusterPrometheusNamespace string

	projects        monitoringcontrollers.ProjectController
	projectCache    monitoringcontrollers.ProjectCache
	prometheuses    prometheusoperatorcontrollers.PrometheusController
	alertmanagers   prometheusoperatorcontrollers.AlertmanagerController
	prometheusrules prometheusoperatorcontrollers.PrometheusRuleController
	podmonitors     prometheusoperatorcontrollers.PodMonitorController
	namespaces      corecontrollers.NamespaceController
	namespaceCache  corecontrollers.NamespaceCache
}

func Register(
	ctx context.Context,
	systemNamespace, clusterPrometheusName, clusterPrometheusNamespace string,
	apply apply.Apply,
	projects monitoringcontrollers.ProjectController,
	projectCache monitoringcontrollers.ProjectCache,
	prometheuses prometheusoperatorcontrollers.PrometheusController,
	alertmanagers prometheusoperatorcontrollers.AlertmanagerController,
	prometheusrules prometheusoperatorcontrollers.PrometheusRuleController,
	podmonitors prometheusoperatorcontrollers.PodMonitorController,
	namespaces corecontrollers.NamespaceController,
	namespaceCache corecontrollers.NamespaceCache,
) {

	h := &handler{
		systemNamespace:            systemNamespace,
		clusterPrometheusName:      clusterPrometheusName,
		clusterPrometheusNamespace: clusterPrometheusNamespace,
		projects:                   projects,
		projectCache:               projectCache,
		prometheuses:               prometheuses,
		alertmanagers:              alertmanagers,
		prometheusrules:            prometheusrules,
		podmonitors:                podmonitors,
		namespaces:                 namespaces,
		namespaceCache:             namespaceCache,
	}

	monitoringcontrollers.RegisterProjectGeneratingHandler(ctx,
		projects,
		apply.WithCacheTypes(
			prometheuses,
			alertmanagers,
			prometheusrules,
			podmonitors,
			// namespaces,
		),
		"",
		"project-registration",
		h.OnChange,
		nil)

	relatedresource.Watch(ctx, "sync-project-resources", h.resolveProjectOwned, projects,
		prometheuses, alertmanagers, prometheusrules, podmonitors, namespaces)

	relatedresource.Watch(ctx, "sync-project-namespaces", h.resolveNotProjectOwned, projects,
		namespaces)
}

func (h *handler) resolveProjectOwned(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
	return relatedresource.OwnerResolver(true, monitoring.SchemeGroupVersion.String(), "Project")(namespace, name, obj)
}

func (h *handler) resolveNotProjectOwned(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
	var keys []relatedresource.Key
	ns, ok := obj.(*v1.Namespace)
	if !ok {
		return nil, nil
	}
	if name == h.systemNamespace || name == h.clusterPrometheusNamespace || h.isProjectNamespace(ns) {
		// No project should select the clusterPrometheus namespac or a project namespace
		return nil, nil
	}

	projects, err := h.projectCache.List(h.systemNamespace, labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, project := range projects {
		selector, err := metav1.LabelSelectorAsSelector(project.Spec.Selector)
		if err != nil {
			// even if one project is wrong, emit an error and continue with the other projects
			logrus.Errorf("could not parse selector from project %s", project.Name)
			continue
		}
		if selector.Matches(labels.Set(ns.Labels)) {
			keys = append(keys, relatedresource.Key{
				Namespace: h.systemNamespace,
				Name:      project.Name,
			})
		}
	}
	return keys, err
}

func (h *handler) OnChange(project *monitoring.Project, status monitoring.ProjectStatus) ([]runtime.Object, monitoring.ProjectStatus, error) {
	var objs []runtime.Object
	status.ProjectNamespace = project.Name
	status.ClusterPrometheus = fmt.Sprintf("%s/%s", h.clusterPrometheusNamespace, h.clusterPrometheusName)
	status.Namespaces = h.findNamespaces(project.Spec.Selector)
	return objs, status, nil
}

func (h *handler) findNamespaces(labelSelector *metav1.LabelSelector) []string {
	if labelSelector == nil {
		return nil
	}
	nsLabelSelector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil
	}
	namespaces, err := h.namespaceCache.List(nsLabelSelector)
	if err != nil {
		return nil
	}
	var namespaceList []string
	for _, ns := range namespaces {
		if ns.Name == h.systemNamespace || ns.Name == h.clusterPrometheusNamespace || h.isProjectNamespace(ns) {
			continue
		}
		namespaceList = append(namespaceList, ns.Name)
	}
	return namespaceList
}

func (h *handler) isProjectNamespace(ns *v1.Namespace) bool {
	keys, err := h.resolveProjectOwned("", ns.Name, ns)
	return err != nil && len(keys) > 0
}
