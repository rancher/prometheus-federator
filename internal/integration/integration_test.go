package integration

import (
	. "github.com/onsi/ginkgo/v2"
	helm_locker "github.com/rancher/prometheus-federator/internal/helm-locker"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/project"
	"github.com/rancher/prometheus-federator/internal/test"
	"github.com/rancher/prometheus-federator/pkg/instrumentation"
)

func init() {
	instrumentation.InitTracing("prometheus-federator-integration-tests")
}

const projectIdLabel = "field.cattle.io/projectId"
const overrideProjectLabel = "x.y.z/projectId"

// Initialize clients, object trackers and contexts used by the tests
var _ = BeforeSuite(test.Setup)

var _ = Describe("Prometheus Federator integration tests", Ordered, func() {
	Describe("HPO/NamespaceController/Single", Ordered, namespace.SingleNamespaceTest())
	Describe("HPO/NamespaceController/InvalidYaml", Ordered, namespace.MultiNamespaceTest(
		"cattle-helm-system",
		common.Options{
			OperatorOptions: common.OperatorOptions{
				HelmAPIVersion:   "v1",
				SystemNamespaces: []string{"kube-system"},
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
		},
		"values?",
		"questions?",
		"project-id-1",
	))
	Describe("HPO/NamespaceController/ValidYaml", Ordered, namespace.MultiNamespaceTest(
		"cattle-helm-system",
		common.Options{
			OperatorOptions: common.OperatorOptions{
				HelmAPIVersion:   "v1",
				SystemNamespaces: []string{"kube-system"},
			},
			RuntimeOptions: common.RuntimeOptions{
				Namespace:                     "helm-project-operator-test",
				ControllerName:                "helm-project-operator-test",
				HelmJobImage:                  "rancher/klipper-helm:v0.9.4-build20250113",
				ProjectLabel:                  overrideProjectLabel,
				ProjectReleaseLabelValue:      "p-test-release",
				DisableEmbeddedHelmLocker:     true,
				DisableEmbeddedHelmController: true,
			},
		},
		validValuesYaml,
		emptyQuestions,
		"project-id-2",
	))
	Describe("HPO/ProjectController", Ordered, project.ProjectControllerTest(
		"cattle-helm-system",
		common.Options{
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
		map[string]interface{}{
			"contents2": "alwaysOverriden",
		},
		projectGetter(
			[]string{},
			[]string{},
			map[string][]string{},
		),
	))
	Describe("HelmLocker/e2e", Ordered, helm_locker.E2eTest())
})

const validValuesYaml = `
enabled:
	true
`

const emptyQuestions = ``
