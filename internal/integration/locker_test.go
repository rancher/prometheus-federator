package integration

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/schemes"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helm_locker "github.com/rancher/prometheus-federator/internal/helm-locker"
	"github.com/rancher/prometheus-federator/internal/helm-locker/controllers/release"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/prometheus-federator/internal/test"
)

const helmLockerControllerNs = "cattle-helm-locker-system"
const controllerName = "helm-locker-test"

var _ = Describe("Helm Locker", func() {
	// OncePerOrdered ensures this is run once for each downstream node, without requiring
	// that they are run one by one
	BeforeEach(OncePerOrdered, func() {
		startEmbeddedHelmController()
	})

	AfterEach(OncePerOrdered, func() {
		// optional cleanup after each node
	})

	// Here we can run the following tests in parallel, by specifying --procs=N using ginkgo
	Describe("HelmLocker/e2e", Ordered, helm_locker.E2eTest(helmLockerControllerNs, controllerName, "node1"))
})

func startEmbeddedHelmController() {
	ti := test.GetTestInterface()
	appCtx, err := setup.NewAppContext(ti.ClientConfig(), helmLockerControllerNs, common.Options{})
	Expect(err).To(Succeed())
	ctxca, ca := context.WithCancel(ti.Context())

	recorder := appCtx.EventBroadcaster.NewRecorder(schemes.All, corev1.EventSource{
		Component: "helm-project-operator",
		Host:      "node1",
	})

	DeferCleanup(func() {
		ca()
	})

	release.Register(ti.Context(),
		helmLockerControllerNs,
		controllerName,
		appCtx.HelmLocker.HelmRelease(),
		appCtx.HelmLocker.HelmRelease().Cache(),
		appCtx.Core.Secret(),
		appCtx.Core.Secret().Cache(),
		appCtx.K8s,
		appCtx.ObjectSetRegister,
		appCtx.ObjectSetHandler,
		recorder,
	)

	Expect(appCtx.Start(ctxca)).To(Succeed())
}
