package integration

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/prometheus-federator/internal/test"
)

func TestIntegration(t *testing.T) {
	SetDefaultEventuallyTimeout(60 * time.Second)
	SetDefaultEventuallyPollingInterval(50 * time.Millisecond)
	SetDefaultConsistentlyDuration(5 * time.Second)
	SetDefaultConsistentlyPollingInterval(50 * time.Millisecond)
	RegisterFailHandler(Fail)

	// Dev : to debug specific suites, use the following
	// suiteConfig, reporterConfig := GinkgoConfiguration()
	// // suiteConfig.LabelFilter = "DEBUG"
	// suiteConfig.FocusStrings = []string{"HPO/Namespace/Init"}
	// RunSpecs(t, "Integration Suite", suiteConfig, reporterConfig)

	// comment out the following line to run only the specific suite
	RunSpecs(t, "Integration Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// setup once across all processes
	test.SetupOnce()
	return []byte{}
}, func(_ []byte) {
	// setup across all downstream processes
	test.Setup()
})
