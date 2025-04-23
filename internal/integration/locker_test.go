package integration

import (
	"context"

	"github.com/google/uuid"
	"github.com/rancher/wrangler/v3/pkg/schemes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helm_locker "github.com/rancher/prometheus-federator/internal/helm-locker"
	"github.com/rancher/prometheus-federator/internal/helm-locker/controllers/release"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/prometheus-federator/internal/test"
)

var _ = Describe("HelmLocker", func() {
	// This tests the helm-locker is correctly namespaced and can run in parallel with another non-conflicting helm-locker
	Describe("| HelmLocker/e2e1 |", HelmLockerTestSetup("HelmLocker/e2e1"))
	Describe("| HelmLocker/e2e2 |", HelmLockerTestSetup("HelmLocker/e2e2"))
})

const helmLockerControllerNs = "cattle-helm-locker-system"

func HelmLockerTestSetup(name string) func() {
	return func() {
		var (
			testUUID       string
			ns             string
			controllerName string
			nodeName       string
		)

		BeforeEach(OncePerOrdered, func() {
			testUUID = uuid.New().String()
			ns = helmLockerControllerNs + "-" + testUUID
			controllerName = "locker-" + testUUID
			nodeName = "node-" + testUUID

			// create required system namespace
			createNs(ns)
			// start controller
			startEmbeddedHelmLocker(ns, controllerName)
		})

		AfterEach(OncePerOrdered, func() {

		})

		Describe(name, Ordered, helm_locker.E2eTest(func() helm_locker.TestInfo {
			return helm_locker.TestInfo{
				SystemNamespace: ns,
				NodeName:        nodeName,
				ControllerName:  controllerName,
				UUID:            testUUID,
			}
		}))
	}
}

func createNs(name string) {
	ti := test.GetTestInterface()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	testSetupUid := uuid.New().String()
	ti.ObjectTracker().ObjectTracker(testSetupUid).Add(ns)
	Expect(ti.K8sClient().Create(
		ti.Context(),
		ns,
	)).To(Succeed())

	DeferCleanup(func() {
		ti.ObjectTracker().ObjectTracker(testSetupUid).DeleteAll()
	})
}

// starts a helm-locker controller process, and the hooks to clean it up
func startEmbeddedHelmLocker(
	systemNamespace,
	controllerName string,
) {
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
		systemNamespace,
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
