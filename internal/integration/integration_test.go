package integration

import (
	. "github.com/onsi/ginkgo/v2"
	helm_locker "github.com/rancher/prometheus-federator/internal/helm-locker"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/test"
	"github.com/rancher/prometheus-federator/pkg/instrumentation"
)

func init() {
	instrumentation.InitTracing("prometheus-federator-integration-tests")
}

// Initialize clients, object trackers and contexts used by the tests
var _ = BeforeSuite(test.Setup)

var _ = Describe("Prometheus Federator integration tests", Ordered, func() {
	Describe("HPO/SingleNamespaceController", Ordered, namespace.SingleNamespaceTest())
	Describe("HPO/MultiNamespaceController", Ordered, namespace.MultiNamespaceTest())
	Describe("HelmLocker/e2e", Ordered, helm_locker.E2eTest())
})
