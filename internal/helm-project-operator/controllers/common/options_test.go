package common

import "testing"

func TestOptionsValidateChartSource(t *testing.T) {
	t.Parallel()

	base := Options{
		OperatorOptions: OperatorOptions{
			HelmAPIVersion: "monitoring.cattle.io/v1alpha1",
			ReleaseName:    "monitoring",
		},
	}

	tests := []struct {
		name    string
		options Options
		wantErr bool
	}{
		{
			name: "embedded chart content remains valid",
			options: func() Options {
				opts := base
				opts.ChartContent = "embedded-chart"
				return opts
			}(),
		},
		{
			name: "managed chart reference is valid without embedded chart",
			options: func() Options {
				opts := base
				opts.ManagedChartName = "rancher-project-monitoring"
				opts.ManagedChartRepo = "https://charts.example.test"
				opts.ManagedChartVersion = "1.2.3"
				return opts
			}(),
		},
		{
			name: "chart configuration is required",
			options: func() Options {
				return base
			}(),
			wantErr: true,
		},
		{
			name: "partial managed chart reference is rejected",
			options: func() Options {
				opts := base
				opts.ManagedChartName = "rancher-project-monitoring"
				opts.ManagedChartRepo = "https://charts.example.test"
				return opts
			}(),
			wantErr: true,
		},
		{
			name: "managed chart repository credentials are valid with a complete reference",
			options: func() Options {
				opts := base
				opts.ManagedChartName = "rancher-project-monitoring"
				opts.ManagedChartRepo = "https://charts.example.test"
				opts.ManagedChartVersion = "1.2.3"
				opts.ManagedChartAuthSecret = "project-monitoring-chart-repo"
				return opts
			}(),
		},
		{
			name: "repository credentials require a complete managed chart reference",
			options: func() Options {
				opts := base
				opts.ChartContent = "embedded-chart"
				opts.ManagedChartAuthSecret = "project-monitoring-chart-repo"
				return opts
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.options.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no validation error, got %v", err)
			}
		})
	}
}
