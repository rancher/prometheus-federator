package crds_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/helmcommon/pkg/crds"
	"github.com/rancher/prometheus-federator/internal/helmcommon/pkg/test/apis"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/samber/lo"
)

var _ = Describe("CRD Suite tests", Label("unit"), func() {
	When("we we use a CRD applier", func() {
		It("should list the CRD definitions it is configured with", func() {
			crdTracker := crds.NewCRDTracker([]crd.CRD{
				crds.StructToCRD(
					"A.test.cattle.io",
					&apis.A{},
					nil,
				),
			})

			Expect(crdTracker.List()).To(HaveLen(1))
		})

		It("should create v1beta1 CRD definitions", func() {
			crdTracker := crds.NewCRDTracker([]crd.CRD{
				crds.StructToCRD(
					"A.test.cattle.io",
					&apis.A{},
					nil,
				)},
				crds.WithV1beta1(true),
			)

			objs, err := crdTracker.Objects()
			Expect(err).To(Succeed())
			Expect(objs).To(HaveLen(1))
		})

		It("should create CRD definitions", func() {
			crdTracker := crds.NewCRDTracker([]crd.CRD{
				crds.StructToCRD(
					"A.test.cattle.io",
					&apis.A{},
					nil,
				)},
			)
			objs, err := crdTracker.Objects()
			Expect(err).To(Succeed())
			Expect(objs).To(HaveLen(1))
		})

		It("should apply extra annotations to CRDs", func() {
			const key, value = "cattle.io/managed-by", "prometheus-federator"
			crdTracker := crds.NewCRDTracker([]crd.CRD{
				crds.StructToCRD(
					"A.test.cattle.io",
					&apis.A{},
					nil,
				)},
				// TODO : this doesn't capture a pointer, i think we would want this to trivially
				// edit a CRD
				crds.WithCustomizeF(
					func(c crd.CRD) crd.CRD {
						c.Annotations = lo.Assign(
							map[string]string{
								key: value,
							},
							c.Annotations,
						)
						return c
					},
				),
			)

			retCrds := crdTracker.List()
			Expect(retCrds).To(HaveLen(1))

			for _, crd := range retCrds {
				Expect(crd.Annotations).To(HaveKeyWithValue(key, value))
			}
		})
	})
})
