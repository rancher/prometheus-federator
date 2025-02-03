package main_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	k3shelmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	dockerparse "github.com/novln/docker-parser"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	lockerv1alpha1 "github.com/rancher/prometheus-federator/internal/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	env "github.com/caarlos0/env/v11"
	"github.com/kralicky/kmatch"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestE2e(t *testing.T) {
	SetDefaultEventuallyTimeout(30 * time.Second)
	SetDefaultEventuallyPollingInterval(50 * time.Millisecond)
	SetDefaultConsistentlyDuration(1 * time.Second)
	SetDefaultConsistentlyPollingInterval(50 * time.Millisecond)
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var (
	k8sClient client.Client
	cfg       *rest.Config
	testCtx   context.Context
	clientSet *kubernetes.Clientset

	clientC clientcmd.ClientConfig
)

type TestSpec struct {
	Kubeconfig string `env:"KUBECONFIG,required"`
	HpoImage   string `env:"IMAGE,required"`

	image *dockerparse.Reference
}

func (t *TestSpec) Validate() error {
	var errs []error
	im, err := dockerparse.Parse(t.HpoImage)
	if err != nil {
		errs = append(errs, err)
	}
	t.image = im
	if _, err := os.Stat(t.Kubeconfig); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

var (
	ts = TestSpec{}
)

var _ = BeforeSuite(func() {
	Expect(env.Parse(&ts)).To(Succeed(), "Could not parse test spec from environment variables")
	Expect(ts.Validate()).To(Succeed(), "Invalid input e2e test spec")
	ctxCa, ca := context.WithCancel(context.Background())
	DeferCleanup(func() {
		ca()
	})

	niCfg := kubeconfig.GetNonInteractiveClientConfig(ts.Kubeconfig)
	restConfig, err := niCfg.ClientConfig()
	Expect(err).To(Succeed())
	testCtx = ctxCa
	newCfg, err := config.GetConfig()
	cfg = newCfg
	Expect(err).NotTo(HaveOccurred(), "Could not initialize kubernetes client config")
	newClientset, err := kubernetes.NewForConfig(restConfig)
	Expect(err).To(Succeed(), "Could not initialize kubernetes clientset")
	clientSet = newClientset

	newK8sClient, err := client.New(cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred(), "Could not initialize kubernetes client")
	k8sClient = newK8sClient
	v1alpha1.AddToScheme(k8sClient.Scheme())
	k3shelmv1.AddToScheme(k8sClient.Scheme())
	lockerv1alpha1.AddToScheme(k8sClient.Scheme())
	kmatch.SetDefaultObjectClient(k8sClient)
})
