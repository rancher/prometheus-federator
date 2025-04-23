package test

import (
	"context"
	"errors"
	"os"

	k3shelmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	lockerv1alpha1 "github.com/rancher/prometheus-federator/internal/helm-locker/apis/helm.cattle.io/v1alpha1"
	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/sirupsen/logrus"

	env "github.com/caarlos0/env/v11"
	"github.com/kralicky/kmatch"
	commoncrds "github.com/rancher/prometheus-federator/internal/helmcommon/pkg/crds"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	SystemNamespaceCollection = "system-namespaces"
)

var (
	globalTestInterface TestInterface
)

func GetTestInterface() TestInterface {
	if globalTestInterface == nil {
		panic("TestInterface not initialized")
	}
	return globalTestInterface
}

func setTestInterface(ti TestInterface) {
	globalTestInterface = ti
	Expect(globalTestInterface).NotTo(BeNil())
	Expect(globalTestInterface.K8sClient()).NotTo(BeNil())
	Expect(globalTestInterface.RestConfig()).NotTo(BeNil())
	Expect(globalTestInterface.Context()).NotTo(BeNil())
	Expect(globalTestInterface.ClientSet()).NotTo(BeNil())
	Expect(globalTestInterface.ObjectTracker()).NotTo(BeNil())
	Expect(globalTestInterface.ClientConfig()).NotTo(BeNil())
}

type TestInterface interface {
	K8sClient() client.Client
	RestConfig() *rest.Config
	Context() context.Context
	ClientSet() *kubernetes.Clientset
	ObjectTracker() ObjectTrackerBroker
	ClientConfig() clientcmd.ClientConfig
}

type testInterfaceImpl struct {
	k8sClient client.Client
	cfg       *rest.Config
	testCtx   context.Context
	clientSet *kubernetes.Clientset
	o         ObjectTrackerBroker

	clientC clientcmd.ClientConfig
}

func (t *testInterfaceImpl) K8sClient() client.Client {
	return t.k8sClient
}

func (t *testInterfaceImpl) RestConfig() *rest.Config {
	return t.cfg
}

func (t *testInterfaceImpl) Context() context.Context {
	return t.testCtx
}

func (t *testInterfaceImpl) ClientSet() *kubernetes.Clientset {
	return t.clientSet
}

func (t *testInterfaceImpl) ObjectTracker() ObjectTrackerBroker {
	return t.o
}

func (t *testInterfaceImpl) ClientConfig() clientcmd.ClientConfig {
	return t.clientC
}

type TestSpec struct {
	Kubeconfig     string `env:"KUBECONFIG,required"`
	DisableCleanup bool   `env:"DISABLE_CLEANUP"`
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

var (
	ts = TestSpec{}
)

func Setup() {
	if os.Getenv("INTEGRATION") != "true" {
		Skip("Skipping integration tests, use export INTEGRATION=true to run them")
	}
	Expect(env.Parse(&ts)).To(Succeed(), "Could not parse test spec from environment variables")
	Expect(ts.Validate()).To(Succeed(), "Invalid input e2e test spec")
	ctxCa, ca := context.WithCancel(context.Background())
	DeferCleanup(func() {
		ca()
	})
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(GinkgoWriter)

	niCfg := kubeconfig.GetNonInteractiveClientConfig(ts.Kubeconfig)
	clientC := niCfg
	restConfig, err := niCfg.ClientConfig()
	Expect(err).To(Succeed())
	newCfg, err := config.GetConfig()
	cfg := newCfg
	Expect(err).NotTo(HaveOccurred(), "Could not initialize kubernetes client config")
	newClientset, err := kubernetes.NewForConfig(restConfig)
	Expect(err).To(Succeed(), "Could not initialize kubernetes clientset")
	clientSet := newClientset

	newK8sClient, err := client.New(cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred(), "Could not initialize kubernetes client")
	k8sClient := newK8sClient
	var factoryF func() ObjectTracker
	if ts.DisableCleanup {
		factoryF = NewNoopObjectTracker
	} else {
		factoryF = func() ObjectTracker {
			return NewObjectTracker(ctxCa, k8sClient)
		}
	}
	o := NewDefaultObjectTrackerBroker(factoryF)

	DeferCleanup(func() {
		o.DeleteAll()
	})

	managedCrds := common.ManagedCRDsFromRuntime(common.RuntimeOptions{
		DisableEmbeddedHelmLocker:     false,
		DisableEmbeddedHelmController: false,
	})

	Expect(commoncrds.CreateFrom(ctxCa, restConfig, managedCrds)).To(Succeed(), "Failed to create required CRDs for e2e testing")

	testInterface := &testInterfaceImpl{
		testCtx:   ctxCa,
		cfg:       restConfig,
		k8sClient: k8sClient,
		clientSet: clientSet,
		o:         o,
		clientC:   clientC,
	}

	setTestInterface(testInterface)
	v1alpha1.AddToScheme(k8sClient.Scheme())
	k3shelmv1.AddToScheme(k8sClient.Scheme())
	lockerv1alpha1.AddToScheme(k8sClient.Scheme())
	kmatch.SetDefaultObjectClient(k8sClient)
}
