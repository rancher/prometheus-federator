package project

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/prometheus-federator/internal/test"
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
				By("verifying registed role bindings")

				By("starting all controllers")
				ctxca, ca := context.WithCancel(t.Context())
				stopController = ca
				go appCtx.Start(ctxca)
			})
		})

		When("the controller is running", func() {
			It("should do something", func() {
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
