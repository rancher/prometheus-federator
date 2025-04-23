package integration

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/project"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/prometheus-federator/internal/test"
)

const helmProjectControllerNs = "cattle-helm-proj-system"

var _ = Describe("HPO/Project", func() {
	Describe("HPO/Project/TODO", HelmProjectTestSetup("HPO/Project/TODO", TestConfig{
		Opts: common.Options{
			OperatorOptions: common.OperatorOptions{
				HelmAPIVersion: "v1",
				ReleaseName:    "test-1",
				// TODO : set this to a valid chart content
				ChartContent: "",
			},
			RuntimeOptions: common.RuntimeOptions{
				ProjectLabel: projectIdLabel,
			},
		},
	}))
	// Describe("HPO/Project/TODO2", HelmProjectTestSetup("HPO/Project/TODO2", TestConfig{}))
})

type TestConfig struct {
	Opts common.Options
}

func HelmProjectTestSetup(name string, config TestConfig) func() {
	return func() {
		var (
			testUUID          string
			projectTestConfig project.TestConfig
		)

		BeforeEach(OncePerOrdered, func() {
			testUUID = uuid.New().String()
			ns := helmProjectControllerNs + "-" + testUUID
			controllerName := "project-" + testUUID
			// nodeName := "node-" + testUUID
			dummyRegistrationNs := fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, "prog-"+testUUID)
			targetNs := fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, "target-"+testUUID+"-1")
			targetNs2 := fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, "target-"+testUUID+"-2")

			projectGetter := projectGetter(
				[]string{dummyRegistrationNs},
				[]string{"kube-system"},
				map[string][]string{
					dummyRegistrationNs: {targetNs, targetNs2},
				},
			)

			createNs(ns)
			createNs(dummyRegistrationNs)
			createNs(targetNs)
			createNs(targetNs2)

			startEmbeddedHelmProject(ns, controllerName, projectGetter)

			projectTestConfig = project.TestConfig{
				TestUUID:                   testUUID,
				Opts:                       config.Opts,
				DummyRegistrationNamespace: dummyRegistrationNs,
				TestProjectGetter:          projectGetter,
			}
		})

		AfterEach(OncePerOrdered, func() {

		})

		Describe(name, Ordered, project.ProjectControllerTest(func() project.TestConfig {
			return projectTestConfig
		}))
	}
}

var _ = Describe("HPO/FAKEprojectController", func() {
	// OncePerOrdered ensures this is run once for each downstream node, without requiring
	// that they are run one by one
	BeforeEach(OncePerOrdered, func() {
	})

	AfterEach(OncePerOrdered, func() {
		// optional cleanup
	})

	Describe("HPO/ProjectController", Ordered, project.ProjectControllerTest(
		func() project.TestConfig {
			return project.TestConfig{}
		},
		// "cattle-helm-system",
		// common.Options{
		// 	OperatorOptions: common.OperatorOptions{
		// 		HelmAPIVersion: "v1",
		// 		ReleaseName:    "test-1",
		// 		// TODO : set this to a valid chart content
		// 		ChartContent: "",
		// 	},
		// 	RuntimeOptions: common.RuntimeOptions{
		// 		ProjectLabel: projectIdLabel,
		// 	},
		// },
		// map[string]interface{}{
		// 	"contents2": "alwaysOverriden",
		// },
		// projectGetter(
		// 	[]string{},
		// 	[]string{},
		// 	map[string][]string{},
		// ),
	))
})

func startEmbeddedHelmProject(
	systemNamespace, controllerName string,
	projectGetter namespace.ProjectGetter,
) {
	opts := common.Options{}

	ti := test.GetTestInterface()
	appCtx, err := setup.NewAppContext(ti.ClientConfig(), systemNamespace, common.Options{})
	Expect(err).To(Succeed())

	ctxca, ca := context.WithCancel(ti.Context())
	DeferCleanup(func() {
		ca()
	})

	valuesOverride := v1alpha1.GenericMap{}

	project.Register(ctxca,
		systemNamespace,
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

	Expect(appCtx.Start(ctxca)).To(Succeed())
}
