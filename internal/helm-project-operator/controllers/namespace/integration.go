package namespace

import (
	"context"
	"fmt"
	"time"

	. "github.com/kralicky/kmatch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/prometheus-federator/internal/helmcommon/pkg/crds"
	"github.com/rancher/prometheus-federator/internal/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SingleNamespaceTest() func() {
	return func() {
		When("we inject the namespace controller test with a delay", func() {
			It("should sleep arbitrarily", func() {
				time.Sleep(5 * time.Second)
				Expect(true).To(BeTrue())
			})
		})
	}
}

func MultiNamespaceTest() func() {
	return func() {
		const (
			testSystemNamespace = "cattle-helm-system"
			projectIdLabel      = "field.cattle.io/projectId"
		)
		const (
			projectDataConfigmap = "v1"
			projectQuestions     = "questions?"
			projectValues        = "values?"
		)

		var (
			DummySystemNamespaces = []string{"kube-system"}
			opts                  = common.Options{
				OperatorOptions: common.OperatorOptions{
					HelmAPIVersion:   projectDataConfigmap,
					SystemNamespaces: DummySystemNamespaces,
				},
				RuntimeOptions: common.RuntimeOptions{
					Namespace:                     "helm-project-operator-test",
					ControllerName:                "helm-project-operator-test",
					HelmJobImage:                  "rancher/klipper-helm:v0.9.4-build20250113",
					ProjectLabel:                  projectIdLabel,
					ProjectReleaseLabelValue:      "p-test-release",
					DisableEmbeddedHelmLocker:     true,
					DisableEmbeddedHelmController: true,
				},
			}
			projectGetter  ProjectGetter
			appCtx         *setup.AppContext
			t              test.TestInterface
			controllerStop context.CancelFunc
		)
		BeforeAll(func() {
			t = test.GetTestInterface()

			opts := common.Options{
				OperatorOptions: common.OperatorOptions{
					HelmAPIVersion:   projectDataConfigmap,
					SystemNamespaces: DummySystemNamespaces,
				},
				RuntimeOptions: common.RuntimeOptions{
					Namespace:                     "helm-project-operator-test",
					ControllerName:                "helm-project-operator-test",
					HelmJobImage:                  "rancher/klipper-helm:v0.9.4-build20250113",
					ProjectLabel:                  projectIdLabel,
					ProjectReleaseLabelValue:      "p-test-release",
					DisableEmbeddedHelmLocker:     true,
					DisableEmbeddedHelmController: true,
				},
			}
			By("creating required CRDs")

			managedCrds := common.ManagedCRDsFromRuntime(opts.RuntimeOptions)
			Expect(len(managedCrds)).To(BeNumerically(">", 0))
			Expect(crds.CreateFrom(t.Context(), t.RestConfig(), managedCrds)).To(Succeed())

			By("setting up the app context")

			a, err := setup.NewAppContext(t.ClientConfig(), testSystemNamespace, common.Options{})
			Expect(err).To(Succeed())
			appCtx = a
		})

		AfterAll(func() {
			By("stopping the controller")
			if controllerStop != nil {
				controllerStop()
			}
			controllerStop()
			By("flusing the current objects in the object tracker")
			t.ObjectTracker().DeleteAll()
		})

		When("we initialize the namespace controller", func() {
			It("should have correctly tracked the project registration namespaces / system namespaces before the reconcilers run", func() {
				// TODO : https://github.com/rancher/prometheus-federator/pull/175
				By("registering the namespace controller")
				projectGetter = Register(
					t.Context(),
					appCtx.Apply,
					testSystemNamespace,
					projectValues,
					projectQuestions,
					opts,
					// watches and generates
					appCtx.Core.Namespace(),
					appCtx.Core.Namespace().Cache(),
					appCtx.Core.ConfigMap(),
					// enqueues
					appCtx.ProjectHelmChart(),
					appCtx.ProjectHelmChart().Cache(),
					appCtx.Dynamic,
				)

				// TODO : create some clear project registration namespaces

				// ...

				// TODO : verifying projectGetter.IsProjectRegistrationNamespace

				// ...
				By("starting the controllers after we make sure trackers are initialized")
				ctxca, ca := context.WithCancel(t.Context())
				controllerStop = ca
				go appCtx.Start(ctxca)

			})
		})
		When("we use the namespace controller", func() {

			Specify("sanity check we have the requirements to run the controller", func() {
				Expect(projectGetter).NotTo(BeNil())
				GinkgoWriter.Write([]byte("Waiting for namespace controller to create the system namespace\n"))
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testSystemNamespace,
					},
				}
				Eventually(Object(ns)).Should(Exist())

				Eventually(projectGetter.IsSystemNamespace(&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-system",
					},
				})).Should(BeTrue())
			})

			It("should do something with project-registration namespaces", func() {
				const projectId = "dummy-project-id"
				dummyRegistrationNamespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, projectId),
						Labels: map[string]string{
							projectIdLabel: projectId,
						},
					},
				}
				t.ObjectTracker().Add(dummyRegistrationNamespace)
				Expect(t.K8sClient().Create(t.Context(), dummyRegistrationNamespace)).To(Succeed())

				By("verifying the project namespace is created")
				Eventually(Object(dummyRegistrationNamespace)).Should(Exist())

				By("verifying the project namespace has the helm values and questions configmap associate with the chart")
				configmap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      projectDataConfigmap,
						Namespace: dummyRegistrationNamespace.Name,
					},
				}
				Eventually(Object(configmap)).Should(ExistAnd(
					HaveData(
						"values.yaml", projectValues,
						"questions.yaml", projectQuestions,
					),
					HaveOwner(dummyRegistrationNamespace),
				))

				By("verifying the project registration namespace is tracked")
				Eventually(projectGetter.IsProjectRegistrationNamespace(dummyRegistrationNamespace)).Should(BeTrue())
			})

			Specify("when we delete the project registration namespace, it should cleanup related resources", func() {
				const projectId = "dummy-project-id"
				dummyRegistrationNamespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, projectId),
						Labels: map[string]string{
							projectIdLabel: projectId,
						},
					},
				}
				Expect(t.K8sClient().Delete(t.Context(), dummyRegistrationNamespace)).To(Succeed())
				By("verifying the registration namespace is deleted")

				Eventually(Object(dummyRegistrationNamespace)).ShouldNot(Exist())
				Consistently(Object(dummyRegistrationNamespace)).ShouldNot(Exist())

				Expect(t.K8sClient().Delete(t.Context(), dummyRegistrationNamespace)).Should(Succeed())

				By("verifying the tracker eventually stops tracking the namespace")
				Eventually(Object(dummyRegistrationNamespace)).ShouldNot(Exist())
				Eventually(projectGetter.IsProjectRegistrationNamespace(dummyRegistrationNamespace)).Should(BeFalse())
				Consistently(projectGetter.IsProjectRegistrationNamespace(dummyRegistrationNamespace)).Should(BeFalse())
			})
		})
	}
}
