package main_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	lockerv1alpha1 "github.com/rancher/prometheus-federator/internal/helm-locker/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	env "github.com/caarlos0/env/v11"
	"github.com/kralicky/kmatch"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestE2e(t *testing.T) {
	SetDefaultEventuallyTimeout(60 * time.Second)
	SetDefaultEventuallyPollingInterval(50 * time.Millisecond)
	SetDefaultConsistentlyDuration(5 * time.Second)
	SetDefaultConsistentlyPollingInterval(50 * time.Millisecond)
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var (
	k8sClient    client.Client
	cfg          *rest.Config
	testCtx      context.Context
	clientSet    *kubernetes.Clientset
	clientCmdCfg clientcmd.ClientConfig
)

var (
	operatorName = "helm-locker-" + uuid.New().String()
)

type TestSpec struct {
	Kubeconfig string `env:"KUBECONFIG,required"`
}

func (t *TestSpec) Validate() error {
	var errs []error
	if _, err := os.Stat(t.Kubeconfig); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

var _ = BeforeSuite(func() {
	ts := TestSpec{}
	Expect(env.Parse(&ts)).To(Succeed(), "Could not parse test spec from environment variables")
	Expect(ts.Validate()).To(Succeed(), "Invalid input e2e test spec")

	ctxCa, ca := context.WithCancel(context.Background())
	DeferCleanup(func() {
		ca()
	})
	clientCmdCfg = kubeconfig.GetNonInteractiveClientConfig(ts.Kubeconfig)

	testCtx = ctxCa
	newCfg, err := config.GetConfig()
	Expect(err).NotTo(HaveOccurred(), "Could not initialize kubernetes client config")
	cfg = newCfg
	newClientset, err := kubernetes.NewForConfig(cfg)
	Expect(err).To(Succeed(), "Could not initialize kubernetes clientset")
	clientSet = newClientset

	newK8sClient, err := client.New(cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred(), "Could not initialize kubernetes client")
	k8sClient = newK8sClient

	lockerv1alpha1.AddToScheme(k8sClient.Scheme())
	apiextensionsv1.AddToScheme(k8sClient.Scheme())
	kmatch.SetDefaultObjectClient(k8sClient)
})
