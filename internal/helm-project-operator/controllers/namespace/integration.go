//go:build integration

package namespace

import (
	"fmt"
	"time"

	. "github.com/kralicky/kmatch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/test"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SingleNamespaceTest() func() {
	// TODO : potential exercise for the avid reader
	return func() {
		When("we do something", func() {
			It("should do something else", func() {
				Expect(true).To(BeTrue())
			})
		})
	}
}

type TestSpecMultiNamespace struct {
	TestUUID          string
	OperatorNamespace string
	Opts              common.Options

	QuestionsYaml   string
	ValuesYaml      string
	TargetProjectId string

	ProjectRegistrationNamespaces []string
	TestProjectGetter             ProjectGetter
}

type TestSpecMultiNamespaceInit struct {
	TestUUID                              string
	ExpectedProjectRegistrationNamespaces []string
	NotProjectRegistrationNamespaces      []string
	TestProjectGetter                     ProjectGetter
}

func MultiNamespaceInitTest(
	testInfoF func() TestSpecMultiNamespaceInit,
) func() {
	return func() {
		var (
			testConfig TestSpecMultiNamespaceInit
		)

		BeforeAll(func() {
			testConfig = testInfoF()
		})

		When("the operator is initialized, before running it", func() {
			It("should have indexed the correct project registration namespaces", func() {
				projectGetter := testConfig.TestProjectGetter
				for _, projectNs := range testConfig.ExpectedProjectRegistrationNamespaces {
					ns := &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: projectNs,
						},
					}
					Eventually(Object(ns)).Should(Exist())
					Eventually(
						projectGetter.IsProjectRegistrationNamespace(ns)).Should(BeTrue(),
						fmt.Sprintf("%s should be tracked by operator as project registration namespace", projectNs),
					)
				}
			})

			It("should ignore invalid project registration namespaces", func() {
				projectGetter := testConfig.TestProjectGetter
				for _, projectNs := range testConfig.NotProjectRegistrationNamespaces {
					ns := &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: projectNs,
						},
					}
					Eventually(Object(ns)).Should(Exist())
					Expect(
						projectGetter.IsProjectRegistrationNamespace(ns)).To(BeFalse(),
						fmt.Sprintf("%s should not be tracked by operator as project registration namespace", projectNs),
					)
				}
			})
		})
	}
}

func MultiNamespaceTest(
	testInfoF func() TestSpecMultiNamespace,
) func() {
	return func() {
		var (
			ti            test.TestInterface
			testInfo      TestSpecMultiNamespace
			o             test.ObjectTracker
			projectGetter ProjectGetter
		)

		BeforeAll(func() {
			ti = test.GetTestInterface()
			testInfo = testInfoF()
			o = ti.ObjectTracker().Scoped(testInfo.TestUUID)
			projectGetter = testInfo.TestProjectGetter
			DeferCleanup(func() {
				o.DeleteAll()
			})
		})

		When("when we run the namespace controller", func() {
			Specify("sanity check we have the requirements to run the controller", func() {
				Expect(projectGetter).NotTo(BeNil())
				GinkgoWriter.Write([]byte("Waiting for namespace controller to create the system namespace\n"))
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testInfo.OperatorNamespace,
					},
				}
				Eventually(Object(ns)).Should(Exist())

				Eventually(projectGetter.IsSystemNamespace(&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-system",
					},
				})).Should(BeTrue())
			})

			It("should have correctly tracked system namespaces", func() {
				for _, systemNs := range testInfo.Opts.OperatorOptions.SystemNamespaces {
					ns := &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: systemNs,
						},
					}
					Eventually(
						projectGetter.IsSystemNamespace(ns)).Should(BeTrue(),
						fmt.Sprintf("%s should be tracked by operator as system namespace", systemNs),
					)
				}
			})
		})

		When("we use the namespace controller", func() {
			It("should do something with project-registration namespaces", func() {
				opts := testInfo.Opts
				dummyRegistrationNamespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, testInfo.TargetProjectId),
						Labels: map[string]string{
							opts.ProjectLabel: testInfo.TargetProjectId,
						},
					},
				}
				o.Add(dummyRegistrationNamespace)
				Expect(ti.K8sClient().Create(ti.Context(), dummyRegistrationNamespace)).To(Succeed())

				By("verifying the project namespace is created")
				Eventually(Object(dummyRegistrationNamespace)).Should(Exist())

				By("verifying the project registration namespace is tracked")
				// FIXME: this was occasionally failing, indicating that a race condition still exists
				closure := func() bool {
					return projectGetter.IsProjectRegistrationNamespace(dummyRegistrationNamespace)
				}
				Eventually(closure).Should(BeTrue())

				By("verifying the project namespace has the helm values and questions configmap associate with the chart")
				configmap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      opts.OperatorOptions.HelmAPIVersion,
						Namespace: dummyRegistrationNamespace.Name,
					},
				}
				Eventually(Object(configmap)).Should(ExistAnd(
					HaveData(
						"values.yaml", testInfo.ValuesYaml,
						"questions.yaml", testInfo.QuestionsYaml,
					),
					HaveOwner(dummyRegistrationNamespace),
				))

			})

			Specify("when we add namespaces to a project", func() {
				By("manually adding namespaces to a project")
				opts := testInfo.Opts
				nss := []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: fmt.Sprintf("project-%s-ns-1", testInfo.TargetProjectId),
							Labels: map[string]string{
								opts.ProjectLabel: testInfo.TargetProjectId,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: fmt.Sprintf("project-%s-ns-2", testInfo.TargetProjectId),
							Labels: map[string]string{
								opts.ProjectLabel: testInfo.TargetProjectId,
							},
						},
					},
				}
				for _, ns := range nss {
					o.Add(ns)
					Expect(ti.K8sClient().Create(ti.Context(), ns)).To(Succeed())
				}

				By("verifying the project getter tracks them correctly")
				Eventually(func() []string {
					tracker, err := projectGetter.GetTargetProjectNamespaces(&v1alpha1.ProjectHelmChart{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, testInfo.TargetProjectId),
						},
					})
					if err != nil {
						return []string{err.Error()}
					}
					return tracker
				}).Should(ConsistOf(
					lo.Map(nss, func(ns *corev1.Namespace, _ int) string {
						return ns.Name
					}),
				))

				Consistently(func() []string {
					tracker, err := projectGetter.GetTargetProjectNamespaces(&v1alpha1.ProjectHelmChart{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, testInfo.TargetProjectId),
						},
					})
					if err != nil {
						return []string{err.Error()}
					}
					return tracker
				}, 1*time.Second, 200*time.Millisecond).Should(ConsistOf(
					lo.Map(nss, func(ns *corev1.Namespace, _ int) string {
						return ns.Name
					}),
				))
			})

			Specify("when we delete the namespaces associated to a project", func() {

				By("manually deleting namespaces associated to a project")
				opts := testInfo.Opts
				nss := []*corev1.Namespace{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: fmt.Sprintf("project-%s-ns-1", testInfo.TargetProjectId),
							Labels: map[string]string{
								opts.ProjectLabel: testInfo.TargetProjectId,
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: fmt.Sprintf("project-%s-ns-2", testInfo.TargetProjectId),
							Labels: map[string]string{
								opts.ProjectLabel: testInfo.TargetProjectId,
							},
						},
					},
				}
				for _, ns := range nss {
					Expect(ti.K8sClient().Delete(ti.Context(), ns)).To(Succeed())
				}

				By("verifying the project getter stops tracking them")
				Eventually(func() []string {
					tracker, err := projectGetter.GetTargetProjectNamespaces(&v1alpha1.ProjectHelmChart{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, testInfo.TargetProjectId),
						},
					})
					if err != nil {
						return []string{err.Error()}
					}
					return tracker
				}).Should(ConsistOf([]string{}))

				Consistently(func() []string {
					tracker, err := projectGetter.GetTargetProjectNamespaces(&v1alpha1.ProjectHelmChart{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, testInfo.TargetProjectId),
						},
					})
					if err != nil {
						return []string{err.Error()}
					}
					return tracker
				}, 1*time.Second, 200*time.Millisecond).Should(ConsistOf([]string{}))
			})

			Specify("when we delete the project registration namespace, it should cleanup related resources", func() {
				opts := testInfo.Opts
				dummyRegistrationNamespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, testInfo.TargetProjectId),
						Labels: map[string]string{
							opts.ProjectLabel: testInfo.TargetProjectId,
						},
					},
				}
				Expect(ti.K8sClient().Delete(ti.Context(), dummyRegistrationNamespace)).To(Succeed())
				By("verifying the registration namespace is deleted")

				Eventually(Object(dummyRegistrationNamespace)).ShouldNot(Exist())
				Consistently(Object(dummyRegistrationNamespace), 1*time.Millisecond*50).ShouldNot(Exist())

				By("verifying the tracker eventually stops tracking the namespace")
				Eventually(projectGetter.IsProjectRegistrationNamespace(dummyRegistrationNamespace)).Should(BeFalse())
				Consistently(projectGetter.IsProjectRegistrationNamespace(dummyRegistrationNamespace), 1*time.Second, 10*time.Millisecond).Should(BeFalse())
			})
		})
	}
}
