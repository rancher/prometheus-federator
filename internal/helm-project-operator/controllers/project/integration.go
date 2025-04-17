package project

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/prometheus-federator/internal/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// . "github.com/kralicky/kmatch"
)

func ProjectControllerTest(
	operatorNamespace string,
	opts common.Options,
	valuesOverride v1alpha1.GenericMap,
	projectGetter namespace.ProjectGetter,
) func() {
	return func() {
		var (
			t              test.TestInterface
			stopController context.CancelFunc
			appCtx         *setup.AppContext
		)
		BeforeAll(func() {
			t = test.GetTestInterface()

			a, err := setup.NewAppContext(t.ClientConfig(), operatorNamespace, opts)
			Expect(err).To(Succeed())
			appCtx = a

		})

		When("the controller is not yet initialized", func() {
			It("should correctly index the rolebinding cache on initialization", func() {
				Register(
					t.Context(),
					operatorNamespace,
					opts,
					valuesOverride,
					appCtx.Apply,
					// watches
					appCtx.ProjectHelmChart(),
					appCtx.ProjectHelmChart().Cache(),
					appCtx.Core.ConfigMap(),
					appCtx.Core.ConfigMap().Cache(),
					appCtx.RBAC.Role(),
					appCtx.RBAC.Role().Cache(),
					appCtx.RBAC.ClusterRoleBinding(),
					appCtx.RBAC.ClusterRoleBinding().Cache(),
					// watches and generates
					appCtx.HelmController.HelmChart(),
					appCtx.HelmLocker.HelmRelease(),
					appCtx.Core.Namespace(),
					appCtx.Core.Namespace().Cache(),
					appCtx.RBAC.RoleBinding(),
					appCtx.RBAC.RoleBinding().Cache(),
					projectGetter,
				)

				// TODO : https://github.com/rancher/prometheus-federator/pull/166
				By("verifying registered role bindings")

				By("starting all controllers")
				ctxca, ca := context.WithCancel(t.Context())
				stopController = ca
				go appCtx.Start(ctxca)
			})
		})

		When("the controller is running", func() {
			Specify("when we create a project helm chart", func() {
				dummyRegistrationNamespace := fmt.Sprintf(
					common.ProjectRegistrationNamespaceFmt,
					"dummy-registration-namespace",
				)
				projectId := "dumb-1"
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: dummyRegistrationNamespace,
						Labels: map[string]string{
							opts.ProjectLabel: projectId,
						},
					},
				}
				// t.ObjectTracker().Add(ns)
				Expect(t.K8sClient().Create(t.Context(), ns)).To(Succeed())

				ph := &v1alpha1.ProjectHelmChart{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hello",
						Namespace: dummyRegistrationNamespace,
					},
					Spec: v1alpha1.ProjectHelmChartSpec{
						HelmAPIVersion: opts.HelmAPIVersion,
						Values: map[string]interface{}{
							"enabled": "true",
						},
					},
				}
				// t.ObjectTracker().Add(ph)
				Expect(t.K8sClient().Create(t.Context(), ph)).To(Succeed())

				Eventually(func() error {
					return nil
				}).Should(Succeed())
			})
		})

		AfterAll(func() {
			if stopController != nil {
				stopController()
			}
			t.ObjectTracker().DeleteAll()
		})

	}
}
