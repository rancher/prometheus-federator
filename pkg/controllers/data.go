package controllers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func addData(systemNamespace, clusterPrometheusNamespace string, appCtx *appContext) error {
	// TBD: Fill in with resources that need to be added on init, such as the Federation PrometheusRule
	return appCtx.Apply.
		WithSetID("prometheus-federator-bootstrap-data").
		WithDynamicLookup().
		WithNoDeleteGVK(schema.GroupVersionKind{
			Group:   corev1.SchemeGroupVersion.Group,
			Version: corev1.SchemeGroupVersion.Version,
			Kind:    "Namespace",
		}).
		ApplyObjects()
}
