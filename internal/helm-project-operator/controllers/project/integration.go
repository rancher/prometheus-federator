//go:build integration

package project

import (
	helmcontrollerv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	. "github.com/kralicky/kmatch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helmlockerv1alpha1 "github.com/rancher/prometheus-federator/internal/helm-locker/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"

	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestSpecCRUD struct {
	TestUUID string
	Opts     common.Options

	DummyRegistrationNamespace string
	// The project ID matching the dummy registration namespace
	ProjectId         string
	TestProjectGetter namespace.ProjectGetter
	SystemNamespace   string
}

func ProjectControllerTest(
	testConfigF func() TestSpecCRUD,
) func() {
	return func() {
		var (
			ti         test.TestInterface
			o          test.ObjectTracker
			testConfig TestSpecCRUD
		)
		BeforeAll(func() {
			ti = test.GetTestInterface()
			o = ti.ObjectTracker().Scoped("project-controller-test")
			testConfig = testConfigF()
			DeferCleanup(func() {
				o.DeleteAll()
			})
		})

		When("the controller is running", func() {
			Specify("Sanity check we have requirements to run the controller", func() {
				Expect(testConfig.TestUUID).NotTo(BeEmpty())
				Expect(testConfig.DummyRegistrationNamespace).NotTo(BeEmpty())
				Expect(testConfig.TestProjectGetter).NotTo(BeNil())
				Expect(testConfig.Opts.HelmAPIVersion).NotTo(BeEmpty())

				Eventually(
					Object(&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testConfig.DummyRegistrationNamespace,
						},
					}),
				).Should(Exist())
				getter := testConfig.TestProjectGetter
				Expect(
					getter.IsProjectRegistrationNamespace(&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testConfig.DummyRegistrationNamespace,
						},
					}),
				).Should(BeTrue())

				Expect(
					getter.GetTargetProjectNamespaces(
						&v1alpha1.ProjectHelmChart{},
					),
				)

				Eventually(
					testConfig.TestProjectGetter.IsProjectRegistrationNamespace(&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: testConfig.DummyRegistrationNamespace,
						},
					}),
				).Should(BeTrue())
			})
		})

		When("we create a project helm chart", func() {
			Specify("the helm chart should move to a valid state", func() {
				opts := testConfig.Opts

				ph := projectHelmChart(
					"hello",
					testConfig.DummyRegistrationNamespace,
					opts,
					map[string]interface{}{
						"enabled": "true",
					},
				)
				o.Add(ph)
				By("checking the release namespace exists")
				releaseName, releaseNamespace := getReleaseNamespaceAndNameRaw(ph, opts)
				GinkgoWriter.Printf("Release name: %s, Release namespace: %s\n", releaseName, releaseNamespace)
				// Eventually(Object(&corev1.Namespace{
				// 	ObjectMeta: metav1.ObjectMeta{
				// 		Name: releaseNamespace,
				// 	},
				// })).Should(Exist())

				// 1. Need to create the namespace controller release configmap

				By("creating the matching release config map")
				// this is the config map that should be created by the namespace controller that is running,
				// however this test isolates the project controller specifically
				// it is created from `getConfigMap` in namespace/resources.go

				// this is a cross-cutting concern between the two controllers, that we should think about
				// how to handle better
				// cfgMap := &corev1.ConfigMap{
				// 	ObjectMeta: metav1.ObjectMeta{
				// 		Name:      releaseName,
				// 		Namespace: releaseNamespace,
				// 		Labels: map[string]string{
				// 			common.HelmProjectOperatorDashboardValuesConfigMapLabel: releaseName,
				// 		},
				// 	},
				// 	Data: map[string]string{
				// 		"data.json": `{"enabled": "true"}`,
				// 	},
				// }
				// o.Add(cfgMap)
				// Expect(ti.K8sClient().Create(ti.Context(), cfgMap)).To(Succeed())
				// Eventually(Object(cfgMap)).Should(Exist())

				// 2. Need to create the dashboard values configmap
				// it is indexed by its own namespace and release label (helm-dashboard-label : releaseName)

				// dashboardCfgMap := &corev1.ConfigMap{
				// 	ObjectMeta: metav1.ObjectMeta{
				// 		Name: "dashboard-values",
				// 		// this should be arbitrary but who knows
				// 		Namespace: testConfig.Opts.SystemNamespaces[0],
				// 		Labels: map[string]string{
				// 			common.HelmProjectOperatorDashboardValuesConfigMapLabel: releaseName,
				// 		},
				// 	},
				// }
				// o.Add(dashboardCfgMap)
				// Expect(ti.K8sClient().Create(ti.Context(), dashboardCfgMap)).To(Succeed())
				// Eventually(Object(dashboardCfgMap)).Should(Exist())

				By("creating the project helm chart")
				Expect(ti.K8sClient().Create(ti.Context(), ph)).To(Succeed())
				Eventually(Object(ph)).Should(Exist())

				By("verifying the status is eventually OK")

				Eventually(func() string {
					ph, err := Object(ph)()
					if err != nil {
						return err.Error()
					}
					return ph.Status.Status

				}).Should(Or(Equal("Deployed"), Equal("WaitingForDashboardValues")))
				// TODO : when valid dashboard values are created, make sure the status moves to Deployed
			})

			Specify("the matching helm chart CR should be managed", func() {
				opts := testConfig.Opts

				ph := projectHelmChart(
					"hello",
					testConfig.DummyRegistrationNamespace,
					opts,
					map[string]interface{}{
						"enabled": "true",
					},
				)
				_, releaseName := getReleaseNamespaceAndNameRaw(ph, opts)
				By("verifying the object is created")
				helmChart := &helmcontrollerv1.HelmChart{
					ObjectMeta: metav1.ObjectMeta{
						Name:      releaseName,
						Namespace: testConfig.SystemNamespace,
					},
				}

				Eventually(Object(helmChart)).Should(Exist())
				By("verifying the object holds the correct values")

			})

			Specify("the matching helm release CR should be created", func() {
				opts := testConfig.Opts

				ph := projectHelmChart(
					"hello",
					testConfig.DummyRegistrationNamespace,
					opts,
					map[string]interface{}{
						"enabled": "true",
					},
				)

				_, releaseName := getReleaseNamespaceAndNameRaw(ph, opts)
				By("verifying the object is created")
				hr := &helmlockerv1alpha1.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      releaseName,
						Namespace: testConfig.SystemNamespace,
					},
				}

				By("verifying the object references the correct helm chart CR")
				// Make sure it references the correct helm
				// and has the correct labels
				// TODO

				Eventually(Object(hr)).Should(Exist())

			})

		})

		When("we delete a project helm chart", func() {
			Specify("the helm chart should be deleted", func() {
				opts := testConfig.Opts

				ph := projectHelmChart(
					"hello",
					testConfig.DummyRegistrationNamespace,
					opts,
					map[string]interface{}{
						"enabled": "true",
					},
				)
				Expect(ti.K8sClient().Delete(ti.Context(), ph)).To(Succeed())
				Eventually(Object(ph)).Should(Not(Exist()))
				Consistently(Object(ph)).Should(Not(Exist()))
			})

			Specify("the matching helm chart CR should be deleted", func() {
				opts := testConfig.Opts

				ph := projectHelmChart(
					"hello",
					testConfig.DummyRegistrationNamespace,
					opts,
					map[string]interface{}{
						"enabled": "true",
					},
				)
				_, releaseName := getReleaseNamespaceAndNameRaw(ph, opts)
				By("verifying the object is created")

				helmChart := &helmcontrollerv1.HelmChart{
					ObjectMeta: metav1.ObjectMeta{
						Name:      releaseName,
						Namespace: testConfig.SystemNamespace,
					},
				}

				Eventually(Object(helmChart)).Should(Not(Exist()))
				Consistently(Object(helmChart)).Should(Not(Exist()))
			})

			Specify("the matching helm release CR should be deleted", func() {
				opts := testConfig.Opts

				ph := projectHelmChart(
					"hello",
					testConfig.DummyRegistrationNamespace,
					opts,
					map[string]interface{}{
						"enabled": "true",
					},
				)

				_, releaseName := getReleaseNamespaceAndNameRaw(ph, opts)
				By("verifying the object is created")
				hr := &helmlockerv1alpha1.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      releaseName,
						Namespace: testConfig.SystemNamespace,
					},
				}
				Eventually(Object(hr)).Should(Not(Exist()))
				Consistently(Object(hr)).Should(Not(Exist()))
			})
		})
	}
}

func projectHelmChart(name, namespace string, opts common.Options, values map[string]interface{}) *v1alpha1.ProjectHelmChart {
	ph := &v1alpha1.ProjectHelmChart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ProjectHelmChartSpec{
			HelmAPIVersion: opts.HelmAPIVersion,
			Values:         values,
		},
	}
	return ph
}
