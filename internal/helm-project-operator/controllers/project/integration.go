package project

import (
	. "github.com/kralicky/kmatch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestConfig struct {
	TestUUID string
	Opts     common.Options

	DummyRegistrationNamespace string
	TestProjectGetter          namespace.ProjectGetter
}

func ProjectControllerTest(
	testConfigF func() TestConfig,
) func() {
	return func() {
		var (
			ti         test.TestInterface
			o          test.ObjectTracker
			testConfig TestConfig
		)
		BeforeAll(func() {
			ti = test.GetTestInterface()
			o = ti.ObjectTracker().ObjectTracker("project-controller-test")
			testConfig = testConfigF()
			DeferCleanup(func() {
				o.DeleteAll()
			})
		})

		// TODO : move this to another suite
		When("the controller is not yet initialized", func() {
			// Skip("skipping this suite")
			It("should correctly index the rolebinding cache on initialization", func() {})
		})

		When("the controller is running", func() {
			Specify("when we create a project helm chart", func() {
				opts := testConfig.Opts

				ph := &v1alpha1.ProjectHelmChart{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello",
						Namespace: testConfig.DummyRegistrationNamespace,
					},
					Spec: v1alpha1.ProjectHelmChartSpec{
						HelmAPIVersion: opts.HelmAPIVersion,
						Values: map[string]interface{}{
							"enabled": "true",
						},
					},
				}
				o.Add(ph)
				Expect(ti.K8sClient().Create(ti.Context(), ph)).To(Succeed())
				Eventually(Object(ph)).Should(Exist())

				By("verifying the matching roles & role bindings are created")

				By("verifying the matching helm release is created")

				By("veriyfing the matching helm chart crd is created ")

				By("verifying the status is eventually OK")

				Eventually(func() string {
					ph, err := Object(ph)()
					if err != nil {
						return err.Error()
					}
					return ph.Status.Status

				}).Should(Equal("Deployed"))
			})
		})
	}
}
