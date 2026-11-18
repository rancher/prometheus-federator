package project

import (
	"testing"

	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetHelmChartUsesEmbeddedChartContentByDefault(t *testing.T) {
	t.Parallel()

	h := &handler{
		systemNamespace: "cattle-monitoring-system",
		opts: common.Options{
			OperatorOptions: common.OperatorOptions{
				ReleaseName:    "monitoring",
				HelmAPIVersion: "monitoring.cattle.io/v1alpha1",
				ChartContent:   "embedded-chart",
			},
		},
	}

	projectHelmChart := &v1alpha1.ProjectHelmChart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-monitoring",
			Namespace: "cattle-project-p1",
		},
		Spec: v1alpha1.ProjectHelmChartSpec{
			HelmAPIVersion: "monitoring.cattle.io/v1alpha1",
		},
	}

	helmChart := h.getHelmChart("p1", "values: {}", projectHelmChart)

	if helmChart.Spec.ChartContent != "embedded-chart" {
		t.Fatalf("expected embedded chart content, got %q", helmChart.Spec.ChartContent)
	}
	if helmChart.Spec.Chart != "project-monitoring-monitoring" {
		t.Fatalf("expected embedded chart name to track release name, got %q", helmChart.Spec.Chart)
	}
	if helmChart.Spec.Repo != "" || helmChart.Spec.Version != "" {
		t.Fatalf("expected no external chart reference, got repo=%q version=%q", helmChart.Spec.Repo, helmChart.Spec.Version)
	}
}

func TestGetHelmChartUsesManagedChartReferenceWhenConfigured(t *testing.T) {
	t.Parallel()

	h := &handler{
		systemNamespace: "cattle-monitoring-system",
		opts: common.Options{
			OperatorOptions: common.OperatorOptions{
				ReleaseName:    "monitoring",
				HelmAPIVersion: "monitoring.cattle.io/v1alpha1",
				ChartContent:   "embedded-chart",
			},
			RuntimeOptions: common.RuntimeOptions{
				ManagedChartName:    "project-monitoring-contract",
				ManagedChartRepo:    "https://charts.example.test",
				ManagedChartVersion: "0.7.0",
			},
		},
	}

	projectHelmChart := &v1alpha1.ProjectHelmChart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-monitoring",
			Namespace: "cattle-project-p1",
		},
		Spec: v1alpha1.ProjectHelmChartSpec{
			HelmAPIVersion: "monitoring.cattle.io/v1alpha1",
		},
	}

	helmChart := h.getHelmChart("p1", "values: {}", projectHelmChart)

	if helmChart.Spec.Chart != "project-monitoring-contract" {
		t.Fatalf("expected managed chart name, got %q", helmChart.Spec.Chart)
	}
	if helmChart.Spec.Repo != "https://charts.example.test" {
		t.Fatalf("expected managed chart repo, got %q", helmChart.Spec.Repo)
	}
	if helmChart.Spec.Version != "0.7.0" {
		t.Fatalf("expected managed chart version, got %q", helmChart.Spec.Version)
	}
	if helmChart.Spec.ChartContent != "" {
		t.Fatalf("expected chart content to be omitted for managed chart reference, got %q", helmChart.Spec.ChartContent)
	}
	if helmChart.Spec.AuthSecret != nil {
		t.Fatalf("expected no chart repo credentials, got %#v", helmChart.Spec.AuthSecret)
	}
}

func TestGetHelmChartSetsManagedChartAuthSecret(t *testing.T) {
	t.Parallel()

	h := &handler{
		systemNamespace: "cattle-monitoring-system",
		opts: common.Options{
			OperatorOptions: common.OperatorOptions{
				ReleaseName:    "monitoring",
				HelmAPIVersion: "monitoring.cattle.io/v1alpha1",
			},
			RuntimeOptions: common.RuntimeOptions{
				ManagedChartName:       "project-monitoring-contract",
				ManagedChartRepo:       "https://charts.example.test",
				ManagedChartVersion:    "0.7.0",
				ManagedChartAuthSecret: "project-monitoring-chart-repo",
			},
		},
	}

	projectHelmChart := &v1alpha1.ProjectHelmChart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-monitoring",
			Namespace: "cattle-project-p1",
		},
		Spec: v1alpha1.ProjectHelmChartSpec{
			HelmAPIVersion: "monitoring.cattle.io/v1alpha1",
		},
	}

	helmChart := h.getHelmChart("p1", "values: {}", projectHelmChart)

	if helmChart.Namespace != "cattle-monitoring-system" {
		t.Fatalf("expected HelmChart in the operator namespace, got %q", helmChart.Namespace)
	}
	if helmChart.Spec.AuthSecret == nil || helmChart.Spec.AuthSecret.Name != "project-monitoring-chart-repo" {
		t.Fatalf("expected auth secret project-monitoring-chart-repo, got %#v", helmChart.Spec.AuthSecret)
	}
}
