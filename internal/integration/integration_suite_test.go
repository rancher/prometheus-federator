package integration

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/test"
	"github.com/rancher/prometheus-federator/pkg/instrumentation"
)

func init() {
	instrumentation.InitTracing("prometheus-federator-integration-tests")
}

func TestIntegration(t *testing.T) {
	SetDefaultEventuallyTimeout(60 * time.Second)
	SetDefaultEventuallyPollingInterval(50 * time.Millisecond)
	SetDefaultConsistentlyDuration(5 * time.Second)
	SetDefaultConsistentlyPollingInterval(50 * time.Millisecond)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

// Initialize clients, object trackers and contexts used by the tests
var _ = BeforeSuite(test.Setup)

// var _ = SynchronizedBeforeSuite(
// 	test.Setup,
// 	func(data []byte) {
// 		// this is invoked after the test suite, this could be things like an audit of which objects
// 		// are created, etc..
// 	})
