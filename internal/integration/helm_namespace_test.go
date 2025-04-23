package integration

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/prometheus-federator/internal/test"
)

const projectIdLabel = "field.cattle.io/projectId"
const overrideProjectLabel = "x.y.z/projectId"

const image = "rancher/klipper-helm:v0.9.4-build20250113"

const helmProjectNamespaceControllerNs = "cattle-helm-ns-system"

type TestNamespaceConfig struct {
	ProjectIdLabel   string
	TargetProjectID  string
	KlipperImage     string
	ReleaseLabel     string
	SystemNamespaces []string
	QuestionsYaml    string
	ValuesYaml       string
}

var _ = Describe("HPO/Namespace", func() {
	// Runs these tests in parallel to make sure we can run scoped namespace controllers without conflict

	Describe("HPO/Namespace/Single", Ordered, namespace.SingleNamespaceTest())
	// Default project ID
	// Has invalid yaml, this is validated before the controller is registered, but we
	// decide to change that the test will fail
	Describe("HPO/Namespace/Multi/Default", Ordered, HelmNamespaceTestSetup("InvalidYaml", TestNamespaceConfig{
		ProjectIdLabel:   projectIdLabel,
		TargetProjectID:  "id-1",
		KlipperImage:     image,
		ReleaseLabel:     "p-test-release",
		SystemNamespaces: []string{"kube-system"},
		ValuesYaml:       "values?",
		QuestionsYaml:    "questions?",
	}))
	// Override project ID
	Describe("HPO/Namespace/Multi/Override", Ordered, HelmNamespaceTestSetup("ValidYaml", TestNamespaceConfig{
		ProjectIdLabel:   overrideProjectLabel,
		TargetProjectID:  "id-2",
		KlipperImage:     image,
		ReleaseLabel:     "p-test-release",
		SystemNamespaces: []string{"kube-system"},
		ValuesYaml:       validValuesYaml,
		QuestionsYaml:    emptyQuestions,
	}))
})

func HelmNamespaceTestSetup(suiteName string, config TestNamespaceConfig) func() {
	return func() {
		var (
			testInfo namespace.TestInfo
		)

		BeforeEach(OncePerOrdered, func() {
			testInfo = namespace.TestInfo{}
			testUUID := uuid.New().String()
			testNs := helmProjectNamespaceControllerNs + "-" + testUUID
			commonOpts := common.Options{
				OperatorOptions: common.OperatorOptions{
					HelmAPIVersion:   "v1",
					SystemNamespaces: config.SystemNamespaces,
				},
				RuntimeOptions: common.RuntimeOptions{
					NamespaceRegistrationRetryMax:              5,
					NamespaceRegistrationWorkers:               10,
					NamespaceRegistrationRetryWaitMilliseconds: 100,
					Namespace:                     testNs,
					ControllerName:                testNs,
					HelmJobImage:                  config.KlipperImage,
					ProjectLabel:                  config.ProjectIdLabel,
					ProjectReleaseLabelValue:      config.ReleaseLabel,
					DisableEmbeddedHelmLocker:     true,
					DisableEmbeddedHelmController: true,
				},
			}
			createNs(testNs)
			projectGetter := startEmbeddedHelmNamespaceController(
				testNs,
				config.ValuesYaml,
				config.QuestionsYaml,
				commonOpts,
			)

			testInfo = namespace.TestInfo{
				TestUUID:          testUUID,
				OperatorNamespace: testNs,
				Opts:              commonOpts,

				QuestionsYaml:   config.QuestionsYaml,
				ValuesYaml:      config.ValuesYaml,
				TargetProjectId: config.TargetProjectID,

				TestProjectGetter: projectGetter,

				// TODO
				ProjectRegistrationNamespaces: []string{},
			}
		})

		AfterEach(OncePerOrdered, func() {

		})

		Describe("", Ordered, namespace.MultiNamespaceTest(
			func() namespace.TestInfo { return testInfo },
		))
	}
}

func startEmbeddedHelmNamespaceController(
	operatorNamespace string,
	valuesYaml, questionsYaml string,
	opts common.Options,
) namespace.ProjectGetter {
	ti := test.GetTestInterface()

	appCtx, err := setup.NewAppContext(ti.ClientConfig(), operatorNamespace, common.Options{})
	Expect(err).To(Succeed(), "Setting up App context failed")

	ctxca, ca := context.WithCancel(ti.Context())
	DeferCleanup(func() {
		ca()
	})
	projectGetter := namespace.Register(
		ctxca,
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

	Expect(appCtx.Start(ctxca)).To(Succeed(), "Starting controller failed")

	return projectGetter
}

const validValuesYaml = `
enabled:
	true
`

const emptyQuestions = ``
