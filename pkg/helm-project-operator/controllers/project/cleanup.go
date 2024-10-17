package project

import (
	"fmt"

	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// initRemoveCleanupLabels removes cleanup labels from all ProjectHelmCharts targeted by this operator
// This gets applied once on startup
func (h *handler) initRemoveCleanupLabels() error {
	namespaceList, err := h.namespaces.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list namespaces to remove cleanup label from all ProjectHelmCharts")
	}
	if namespaceList == nil {
		return nil
	}
	logrus.Infof("Removing cleanup label from all registered ProjectHelmCharts...")
	// ensure all ProjectHelmCharts in every namespace no longer have the cleanup label on them
	for _, ns := range namespaceList.Items {
		projectHelmChartList, err := h.projectHelmCharts.List(ns.Name, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("unable to list ProjectHelmCharts in namespace %s to remove cleanup label", ns.Name)
		}
		if projectHelmChartList == nil {
			continue
		}
		for _, projectHelmChart := range projectHelmChartList.Items {
			shouldManage := h.shouldManage(&projectHelmChart)
			if !shouldManage {
				// not a valid ProjectHelmChart for this operator
				continue
			}
			if projectHelmChart.Labels == nil {
				continue
			}
			_, ok := projectHelmChart.Labels[common.HelmProjectOperatedCleanupLabel]
			if !ok {
				continue
			}
			projectHelmChartCopy := projectHelmChart.DeepCopy()
			delete(projectHelmChartCopy.Labels, common.HelmProjectOperatedCleanupLabel)
			_, err = h.projectHelmCharts.Update(projectHelmChartCopy)
			if err != nil {
				return fmt.Errorf("unable to remove cleanup label from ProjectHelmCharts %s/%s", projectHelmChart.Namespace, projectHelmChart.Name)
			}
		}
	}
	return nil
}
