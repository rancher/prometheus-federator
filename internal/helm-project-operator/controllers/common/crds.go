package common

import (
	lockercrd "github.com/rancher/prometheus-federator/internal/helm-locker/crd"
	helmprojectcrds "github.com/rancher/prometheus-federator/internal/helm-project-operator/crd"
	"github.com/rancher/wrangler/v3/pkg/crd"
)

func ManagedCRDsFromRuntime(options RuntimeOptions) []crd.CRD {
	managedCRDs := helmprojectcrds.Required()
	if !options.DisableEmbeddedHelmLocker {
		managedCRDs = append(managedCRDs, lockercrd.Required()...)
	}
	return managedCRDs
}
