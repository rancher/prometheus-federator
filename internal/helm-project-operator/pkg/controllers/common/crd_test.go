package common_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/controllers/common"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/samber/lo"
)

var _ = Describe("runtime crd tests", Label("unit"), func() {
	When("we construct a list of managed crds", func() {
		It("should manage all crds if all embedded controllers are running", func() {
			managedCrds := common.ManagedCRDsFromRuntime(common.RuntimeOptions{
				DisableEmbeddedHelmLocker:     false,
				DisableEmbeddedHelmController: false,
			})

			ret := lo.Map(managedCrds, func(i crd.CRD, _ int) string {
				return strings.Join([]string{i.GVK.Version, i.GVK.Group, i.GVK.Kind}, ".")
			})
			Expect(ret).To(ConsistOf(
				[]string{
					"v1alpha1.helm.cattle.io.ProjectHelmChart",
					"v1alpha1.helm.cattle.io.HelmRelease",
					"v1.helm.cattle.io.HelmChart",
					"v1.helm.cattle.io.HelmChartConfig",
				},
			))
		})

		It("should not manage helm-controller crds if it does not run the helm-controller itself", func() {
			managedCrds := common.ManagedCRDsFromRuntime(common.RuntimeOptions{
				DisableEmbeddedHelmLocker:     false,
				DisableEmbeddedHelmController: true,
			})

			ret := lo.Map(managedCrds, func(i crd.CRD, _ int) string {
				return strings.Join([]string{i.GVK.Version, i.GVK.Group, i.GVK.Kind}, ".")
			})
			Expect(ret).To(ConsistOf(
				[]string{
					"v1alpha1.helm.cattle.io.ProjectHelmChart",
					"v1alpha1.helm.cattle.io.HelmRelease",
				},
			))
		})

		It("should not manage helm-locker crds if it does not run the helm-locker itself", func() {
			managedCrds := common.ManagedCRDsFromRuntime(common.RuntimeOptions{
				DisableEmbeddedHelmLocker:     true,
				DisableEmbeddedHelmController: false,
			})

			ret := lo.Map(managedCrds, func(i crd.CRD, _ int) string {
				return strings.Join([]string{i.GVK.Version, i.GVK.Group, i.GVK.Kind}, ".")
			})
			Expect(ret).To(ConsistOf(
				[]string{
					"v1alpha1.helm.cattle.io.ProjectHelmChart",
					"v1.helm.cattle.io.HelmChart",
					"v1.helm.cattle.io.HelmChartConfig",
				},
			))
		})
	})
})
