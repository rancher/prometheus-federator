//go:build integration

package integration

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/kralicky/kmatch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/prometheus-federator/internal/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

type TestNamespaceInitConfig struct {
	SystemNamespaces []string
	ReleaseLabel     string
	ProjectIdLabel   string
	// These project IDs will have project registration namespaces created
	ProjectIds []string
	// These project IDs will not have valid associated project registration namespaces
	IgnoreProjectIds []string
}

var _ = Describe("HPO/Namespace", func() {
	// Runs these tests in parallel to make sure we can run scoped namespace controllers without conflict

	Describe("HPO/Namespace/Single", namespace.SingleNamespaceTest())
	// Default project ID
	// Has invalid yaml, this is validated before the controller is registered, but if we
	// decide to change that the test will fail
	Describe("HPO/Namespace/Multi/Default", HelmNamespaceRunOperator("InvalidYaml", TestNamespaceConfig{
		ProjectIdLabel:   projectIdLabel,
		TargetProjectID:  "id-1",
		KlipperImage:     image,
		ReleaseLabel:     "p-test-release",
		SystemNamespaces: []string{"kube-system"},
		ValuesYaml:       "values?",
		QuestionsYaml:    "questions?",
	}))
	// Override project ID
	Describe("HPO/Namespace/Multi/Override", HelmNamespaceRunOperator("ValidYaml", TestNamespaceConfig{
		ProjectIdLabel:   overrideProjectLabel,
		TargetProjectID:  "id-2",
		KlipperImage:     image,
		ReleaseLabel:     "p-test-release",
		SystemNamespaces: []string{"kube-system"},
		ValuesYaml:       validValuesYaml,
		QuestionsYaml:    emptyQuestions,
	}))

	Describe("HPO/Namespace/Init/Default", HelmNamespaceInitializeOperator("Init", TestNamespaceInitConfig{
		SystemNamespaces: []string{"kube-system"},
		ReleaseLabel:     "p-test-release",
		ProjectIdLabel:   projectIdLabel,
		ProjectIds: []string{
			"init-1",
		},
		IgnoreProjectIds: []string{
			"init-ignored-1",
		},
	}))

	Describe("HPO/Namespace/Init/Override", HelmNamespaceInitializeOperator("Init", TestNamespaceInitConfig{
		SystemNamespaces: []string{"kube-system"},
		ReleaseLabel:     "p-test-release",
		ProjectIdLabel:   overrideProjectLabel,
		ProjectIds: []string{
			"init-2",
		},
		IgnoreProjectIds: []string{
			"init-ignored-2",
		},
	}))
})

func HelmNamespaceInitializeOperator(suiteName string, config TestNamespaceInitConfig) func() {
	return func() {
		var (
			testInfo namespace.TestSpecMultiNamespaceInit
		)

		BeforeEach(OncePerOrdered, func() {
			testUUID := uuid.New().String()

			if len(config.ProjectIds) == 0 {
				Fail("No project registration namespaces provided to test config, it is trivially true")
			}
			testNs := helmProjectNamespaceControllerNs + "-" + testUUID
			commonOpts := common.Options{
				OperatorOptions: common.OperatorOptions{
					HelmAPIVersion:   "v1",
					SystemNamespaces: config.SystemNamespaces,
				},
				RuntimeOptions: common.RuntimeOptions{
					NamespaceRegistrationRetryMax:              5,
					NamespaceRegistrationWorkers:               10,
					NamespaceRegistrationRetryWaitMilliseconds: 200,
					Namespace:                     testNs,
					ControllerName:                testNs,
					HelmJobImage:                  image,
					ProjectLabel:                  config.ProjectIdLabel,
					ProjectReleaseLabelValue:      config.ReleaseLabel,
					DisableEmbeddedHelmLocker:     true,
					DisableEmbeddedHelmController: true,
				},
			}
			createNs(testNs)
			expectedRegistrationNs := []string{}
			for _, pid := range config.ProjectIds {
				dummyRegistrationNamespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, pid),
						Labels: map[string]string{
							commonOpts.ProjectLabel: pid,
						},
					},
				}
				createNsFull(dummyRegistrationNamespace)
				expectedRegistrationNs = append(expectedRegistrationNs, dummyRegistrationNamespace.Name)
				Eventually(Object(dummyRegistrationNamespace)).Should(Exist())
			}
			notRegistrationNs := []string{}
			for _, pid := range config.IgnoreProjectIds {
				notDummyRegistrationNamespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, pid),
						// specifically not setting the project label so it should not be registered
					},
				}
				createNsFull(notDummyRegistrationNamespace)
				notRegistrationNs = append(notRegistrationNs, notDummyRegistrationNamespace.Name)
				Eventually(Object(notDummyRegistrationNamespace)).Should(Exist())
			}

			appctx, err := setup.NewAppContext(test.GetTestInterface().ClientConfig(), testNs, commonOpts)
			Expect(err).To(Succeed(), "Setting up App context failed")
			ctxca, ca := context.WithCancel(test.GetTestInterface().Context())
			DeferCleanup(func() {
				ca()
			})

			projectGetter := initEmbeddedHelmNamespaceController(
				appctx,
				ctxca,
				testNs,
				"",
				"",
				commonOpts,
			)

			testInfo = namespace.TestSpecMultiNamespaceInit{
				TestUUID:                              testUUID,
				TestProjectGetter:                     projectGetter,
				ExpectedProjectRegistrationNamespaces: expectedRegistrationNs,
				NotProjectRegistrationNamespaces:      notRegistrationNs,
				ExpectProjectIds:                      config.ProjectIds,
				ExpectNotProjectIds:                   config.IgnoreProjectIds,
			}
		})

		Describe(suiteName, Ordered, namespace.MultiNamespaceInitTest(func() namespace.TestSpecMultiNamespaceInit {
			return testInfo
		}))
	}
}

// HelmNamespaceRunOperator initializes a namespaced helm-project-namespace-operator and runs it.
func HelmNamespaceRunOperator(suiteName string, config TestNamespaceConfig) func() {
	return func() {
		var (
			testInfo namespace.TestSpecMultiNamespace
		)

		BeforeEach(OncePerOrdered, func() {
			testInfo = namespace.TestSpecMultiNamespace{}
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

			testInfo = namespace.TestSpecMultiNamespace{
				TestUUID:          testUUID,
				OperatorNamespace: testNs,
				Opts:              commonOpts,

				QuestionsYaml:   config.QuestionsYaml,
				ValuesYaml:      config.ValuesYaml,
				TargetProjectId: config.TargetProjectID,

				TestProjectGetter: projectGetter,
			}
		})

		// === Here we can register specific tests for a running controller ===
		Describe(suiteName, Ordered, namespace.MultiNamespaceTest(
			func() namespace.TestSpecMultiNamespace { return testInfo },
		))
		// we can extend here to test other functionality of the operator that is not covered in previous tests, that will run in parallel
	}
}

func initEmbeddedHelmNamespaceController(
	appCtx *setup.AppContext,
	ctx context.Context,
	operatorNamespace string,
	valuesYaml, questionsYaml string,
	opts common.Options,
) namespace.ProjectGetter {
	projectGetter := namespace.Register(
		ctx,
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

	return projectGetter
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
	projectGetter := initEmbeddedHelmNamespaceController(
		appCtx,
		ctxca,
		operatorNamespace,
		valuesYaml,
		questionsYaml,
		opts,
	)
	Expect(appCtx.Start(ctxca)).To(Succeed(), "Starting controller failed")
	return projectGetter
}

const validValuesYaml = `
enabled:
	true
`

const emptyQuestions = ``
