package namespace

import (
	"context"
	"fmt"

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
	// TODO : potential exercise for the avid reader
	return func() {
		When("we do something", func() {
			It("should do something else", func() {
				Expect(true).To(BeTrue())
			})
		})
	}
}

func MultiNamespaceTest(
	operatorNamespace string,
	opts common.Options,
	questionsYaml string,
	valuesYaml string,
	targetProjectId string,
) func() {
	return func() {
		var (
			projectGetter  ProjectGetter
			appCtx         *setup.AppContext
			t              test.TestInterface
			controllerStop context.CancelFunc
		)

		BeforeAll(func() {
			t = test.GetTestInterface()

			By("creating required CRDs")

			managedCrds := common.ManagedCRDsFromRuntime(opts.RuntimeOptions)
			Expect(len(managedCrds)).To(BeNumerically(">", 0))
			Expect(crds.CreateFrom(t.Context(), t.RestConfig(), managedCrds)).To(Succeed())

			By("setting up the app context")

			a, err := setup.NewAppContext(t.ClientConfig(), operatorNamespace, common.Options{})
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

			It("should have correctly tracked namespaces", func() {
				// TODO : https://github.com/rancher/prometheus-federator/pull/175
				By("registering the namespace controller")
				projectGetter = Register(
					t.Context(),
					appCtx.Apply,
					operatorNamespace,
					valuesYaml,
					questionsYaml,
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

				By("verifying system namespaces are tracked")
				for _, systemNs := range opts.OperatorOptions.SystemNamespaces {
					ns := &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: systemNs,
						},
					}
					Eventually(projectGetter.IsSystemNamespace(ns)).Should(BeTrue())
				}

				By("verifying project registration namespaces are tracked")

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
						Name: operatorNamespace,
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
				dummyRegistrationNamespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, targetProjectId),
						Labels: map[string]string{
							opts.ProjectLabel: targetProjectId,
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
						Name:      opts.OperatorOptions.HelmAPIVersion,
						Namespace: dummyRegistrationNamespace.Name,
					},
				}
				Eventually(Object(configmap)).Should(ExistAnd(
					HaveData(
						"values.yaml", valuesYaml,
						"questions.yaml", questionsYaml,
					),
					HaveOwner(dummyRegistrationNamespace),
				))

				By("verifying the project registration namespace is tracked")
				Eventually(projectGetter.IsProjectRegistrationNamespace(dummyRegistrationNamespace)).Should(BeTrue())
			})

			Specify("when we delete the project registration namespace, it should cleanup related resources", func() {
				dummyRegistrationNamespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, targetProjectId),
						Labels: map[string]string{
							opts.ProjectLabel: targetProjectId,
						},
					},
				}
				Expect(t.K8sClient().Delete(t.Context(), dummyRegistrationNamespace)).To(Succeed())
				By("verifying the registration namespace is deleted")

				Eventually(Object(dummyRegistrationNamespace)).ShouldNot(Exist())
				Consistently(Object(dummyRegistrationNamespace)).ShouldNot(Exist())

				By("verifying the tracker eventually stops tracking the namespace")
				Eventually(projectGetter.IsProjectRegistrationNamespace(dummyRegistrationNamespace)).Should(BeFalse())
				Consistently(projectGetter.IsProjectRegistrationNamespace(dummyRegistrationNamespace)).Should(BeFalse())
			})
		})
	}
}
