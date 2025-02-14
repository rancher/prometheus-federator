package common

import (
	helmcontrollercrd "github.com/k3s-io/helm-controller/pkg/crd"
	lockercrd "github.com/rancher/prometheus-federator/internal/helm-locker/pkg/crd"
	helmprojectcrds "github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/crd"
)

func ManagedCRDsFromRuntime(options RuntimeOptions) []crd.CRD {
	managedCRDs := helmprojectcrds.Required()
	if !options.DisableEmbeddedHelmLocker {
		managedCRDs = append(managedCRDs, lockercrd.Required()...)
	}
	if !options.DisableEmbeddedHelmController {
		managedCRDs = append(managedCRDs, helmcontrollercrd.List()...)
	}
	return managedCRDs
}
