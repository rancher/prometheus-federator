//go:build integration

package integration

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/project"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/setup"
	"github.com/rancher/prometheus-federator/internal/test"
)

const helmProjectControllerNs = "cattle-helm-proj-system"

var _ = Describe("HPO/Project", func() {
	// tests some expected default values
	Describe("HPO/Project/Default", HelmProjectRunOperator("Default", TestConfig{
		Opts: common.Options{
			OperatorOptions: common.OperatorOptions{
				ReleaseName:    "test-1",
				HelmAPIVersion: "v1",
				// TODO : set this to a valid chart content
				ChartContent: "",
			},
			RuntimeOptions: common.RuntimeOptions{
				ProjectLabel: projectIdLabel,
			},
		},
		ValuesOverride: map[string]interface{}{},
	}))
	// overrides projectIdLabel
	// overrides values.yaml
	Describe("HPO/Project/Override", HelmProjectRunOperator("Override", TestConfig{
		Opts: common.Options{
			OperatorOptions: common.OperatorOptions{
				ReleaseName:    "test-2",
				HelmAPIVersion: "v1",
				// TODO : set this to a valid chart content
				ChartContent: "",
			},
			RuntimeOptions: common.RuntimeOptions{
				ProjectLabel: "x.y.z/projectId",
			},
		},
		ValuesOverride: map[string]interface{}{
			"projectId": "proj-1",
		},
	}))
})

type TestConfig struct {
	Opts           common.Options
	ValuesOverride map[string]interface{}
}

func HelmProjectRunOperator(name string, config TestConfig) func() {
	return func() {
		var (
			testUUID          string
			projectTestConfig project.TestSpecCRUD
		)

		BeforeEach(OncePerOrdered, func() {

			testUUID = uuid.New().String()
			systemNs := helmProjectControllerNs + "-" + testUUID
			projId := "proj-" + testUUID
			dummyRegistrationNs := fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, projId)
			targetNs := fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, "target-"+testUUID+"-1")
			targetNs2 := fmt.Sprintf(common.ProjectRegistrationNamespaceFmt, "target-"+testUUID+"-2")

			projectGetter := projectGetter(
				[]string{dummyRegistrationNs},
				[]string{"kube-system"},
				[]string{
					targetNs, targetNs2,
				},
			)

			createNs(systemNs)
			createNs(dummyRegistrationNs)
			createNs(targetNs)
			createNs(targetNs2)

			startEmbeddedHelmProject(systemNs, projectGetter, config.Opts, config.ValuesOverride)

			projectTestConfig = project.TestSpecCRUD{
				TestUUID:                   testUUID,
				Opts:                       config.Opts,
				DummyRegistrationNamespace: dummyRegistrationNs,
				ProjectId:                  projId,
				TestProjectGetter:          projectGetter,
				SystemNamespace:            systemNs,
			}
		})

		Describe(name, Ordered, project.ProjectControllerTest(func() project.TestSpecCRUD {
			return projectTestConfig
		}))
	}
}

func startEmbeddedHelmProject(
	systemNamespace string,
	projectGetter namespace.ProjectGetter,
	opts common.Options,
	valuesOverride map[string]interface{},
) {
	ti := test.GetTestInterface()
	appCtx, err := setup.NewAppContext(ti.ClientConfig(), systemNamespace, common.Options{})
	Expect(err).To(Succeed())

	ctxca, ca := context.WithCancel(ti.Context())
	DeferCleanup(func() {
		ca()
	})

	project.Register(ctxca,
		systemNamespace,
		opts,
		valuesOverride,
		appCtx.Apply,
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
