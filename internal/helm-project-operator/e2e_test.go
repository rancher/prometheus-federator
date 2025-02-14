package main_test

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/kralicky/kmatch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/mod/semver"

	corev1 "k8s.io/api/core/v1"
	// "sigs.k8s.io/controller-runtime/pkg/client"
	k3shelmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	lockerv1alpha1 "github.com/rancher/prometheus-federator/internal/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/apis/helm.cattle.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"

	// "github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/controllers/common"
	// "github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/operator"
	// "github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/test"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

var (
	//  could be improved to be read from the values.yaml possibly
	cfgName = strings.ReplaceAll("dummy.cattle.io/v1alpha1", "/", ".")
)

// hardcoded labels / annotations / values
const (
	labelProjectID = "field.cattle.io/projectId"
	annoProjectID  = "field.cattle.io/projectId"

	labelHelmProj           = "helm.cattle.io/projectId"
	labelOperatedByHelmProj = "helm.cattle.io/helm-project-operated"
)

// test constants
const (
	// opaque project name
	testProjectName = "p-example"
	// opaque name give to our project helm chart CR
	testPHCName = "project-example-chart"
	// install namespace of the chart
	chartNs = "cattle-helm-system"
	// comes from dummy.go common.OperatorOptions
	releaseName = "dummy"
)

const (
	// DummyHelmAPIVersion is the spec.helmApiVersion corresponding to the dummy example-chart
	DummyHelmAPIVersion = "dummy.cattle.io/v1alpha1"

	// DummyReleaseName is the release name corresponding to the operator that deploys the dummy example-chart
	DummyReleaseName = "dummy"
)

func projectNamespace(project string) string {
	return fmt.Sprintf("cattle-project-%s", project)
}

type helmInstaller struct {
	helmInstallOptions
}

func (h *helmInstaller) build() (*exec.Cmd, error) {
	if h.releaseName == "" {
		return nil, errors.New("helm release name must be set")
	}
	if h.chartRegistry == "" {
		return nil, errors.New("helm chart registry must be set")
	}
	args := []string{
		"upgrade",
		"--install",
	}
	if h.createNamespace {
		args = append(args, "--create-namespace")
	}
	if h.namespace != "" {
		args = append(args, "-n", h.namespace)
	}
	args = append(args, h.releaseName)
	for k, v := range h.values {
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, h.chartRegistry)
	GinkgoWriter.Print(strings.Join(append([]string{"helm"}, append(args, "\n")...), " "))
	cmd := exec.CommandContext(h.ctx, "helm", args...)
	return cmd, nil
}

func newHelmInstaller(opts ...HelmInstallerOption) *helmInstaller {
	h := &helmInstaller{
		helmInstallOptions: helmInstallerDefaultOptions(),
	}
	for _, opt := range opts {
		opt(&h.helmInstallOptions)
	}
	return h
}

func helmInstallerDefaultOptions() helmInstallOptions {
	return helmInstallOptions{
		ctx:             context.Background(),
		createNamespace: false,
		namespace:       "default",
		releaseName:     "helm-project-operator",
		chartRegistry:   "https://charts.helm.sh/stable",
		values:          make(map[string]string),
	}
}

type helmInstallOptions struct {
	ctx             context.Context
	createNamespace bool
	namespace       string
	releaseName     string
	chartRegistry   string
	values          map[string]string
}

type HelmInstallerOption func(*helmInstallOptions)

func WithContext(ctx context.Context) HelmInstallerOption {
	return func(h *helmInstallOptions) {
		h.ctx = ctx
	}
}

func WithCreateNamespace() HelmInstallerOption {
	return func(h *helmInstallOptions) {
		h.createNamespace = true
	}
}

func WithNamespace(namespace string) HelmInstallerOption {
	return func(h *helmInstallOptions) {
		h.namespace = namespace
	}
}

func WithReleaseName(releaseName string) HelmInstallerOption {
	return func(h *helmInstallOptions) {
		h.releaseName = releaseName
	}
}

func WithChartRegistry(chartRegistry string) HelmInstallerOption {
	return func(h *helmInstallOptions) {
		h.chartRegistry = chartRegistry
	}
}

func WithValue(key string, value string) HelmInstallerOption {
	return func(h *helmInstallOptions) {
		if _, ok := h.values[key]; ok {
			panic("duplicate helm value set, likely uninteded behaviour")
		}
		h.values[key] = value
	}
}

var _ = Describe("E2E helm project operator tests", Ordered, Label("kubernetes"), func() {
	BeforeAll(func() {
		By("checking the cluster server version info")
		discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
		Expect(err).To(Succeed(), "Failed to create discovery client")
		serverVersion, err := discoveryClient.ServerVersion()
		Expect(err).To(Succeed(), "Failed to get server version")
		GinkgoWriter.Print(
			fmt.Sprintf("Running e2e tests against Kubernetes distribution %s %s\n",
				strings.TrimPrefix(semver.Build(serverVersion.GitVersion), "+"),
				semver.MajorMinor(serverVersion.GitVersion),
			),
		)
	})

	When("We install the helm project operator", func() {
		// TODO : we need to rework pkg/cli before refactoring the start of the operator.
		// !! We need to be careful to rework them with parity with the way the older rancher/wrangler-cli works
		// !!  which is great but had a lot of default coercion between env variables, structs and other things...
		// TODO : then we can run this "in-tree" instead of importing images and deploying a chart

		It("should install from the latest charts", func() {
			// go func() {
			// 	defer func() {
			// 		// recover from RunOrDie which will always cause a panic on os.Exit
			// 		r := recover()
			// 		if r != nil {
			// 			GinkgoWriter.Write([]byte(fmt.Sprintf("Recovered from panic: %v", r)))
			// 		}
			// 	}()
			// if false {
			// err := operator.Init(testCtx, "cattle-helm-system", clientC, common.Options{
			// 	OperatorOptions: common.OperatorOptions{
			// 		HelmAPIVersion:   DummyHelmAPIVersion,
			// 		ReleaseName:      DummyReleaseName,
			// 		SystemNamespaces: []string{"kube-system"},
			// 		ChartContent:     string(test.TestData("example-chart/example-chart.tgz.base64")),
			// 		Singleton:        false,
			// 	},
			// 	RuntimeOptions: common.RuntimeOptions{
			// 		Namespace:                     "cattle-helm-system",
			// 		DisableEmbeddedHelmController: true,
			// 	},
			// })
			// if err != nil {
			// 	GinkgoWriter.Write([]byte("hello"))
			// }
			// }
			// }()
			ctxT, ca := context.WithTimeout(testCtx, 5*time.Minute)
			defer ca()

			image := strings.TrimPrefix(ts.image.Repository(), "docker.io/")
			helmInstaller := newHelmInstaller(
				WithContext(ctxT),
				WithCreateNamespace(),
				WithNamespace(chartNs),
				WithReleaseName("helm-project-operator"),
				WithChartRegistry("../../examples/helm-project-operator/chart"),
				WithValue("image.registry", "docker.io"),
				WithValue("image.repository", image),
				WithValue("image.tag", ts.image.Tag()),
				WithValue("helmController.enabled", "true"),
			)
			cmd, err := helmInstaller.build()
			Expect(err).To(Succeed())
			session, err := StartCmd(cmd)
			Expect(err).To(Succeed(), "helm install command failed")
			err = session.Wait()
			Expect(err).To(Succeed(), "helm install command failed to exit successfully")
		})

		It("Should create a helm project operator deployment", func() {
			// Skip("implementation detail")
			deploy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "helm-project-operator",
					Namespace: "cattle-helm-system",
				},
			}
			Eventually(Object(deploy)).Should(ExistAnd(
				HaveMatchingContainer(And(
					HaveName("helm-project-operator"),
					HaveImage(fmt.Sprintf("%s:%s", ts.image.Repository(), ts.image.Tag())),
				)),
			))

			Eventually(
				Object(deploy),
				time.Second*90, time.Millisecond*333,
			).Should(HaveSuccessfulRollout())

		})

		When("a project registration namespace is created", func() {
			It("Should create the project registration namespace", func() {
				By("creating the project registration namespace")
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "e2e-hpo",
						Labels: map[string]string{
							// Note : this will be rejected by webhook if rancher/rancher is managing this cluster
							labelProjectID: "p-example",
						},
						Annotations: map[string]string{
							annoProjectID: fmt.Sprintf("local:%s", testProjectName),
						},
					},
				}
				err := k8sClient.Create(testCtx, ns)
				exists := apierrors.IsAlreadyExists(err)
				if !exists {
					Expect(err).To(Succeed(), "Failed to create project registration namespace")
				}
				Eventually(Object(ns)).Should(Exist())

				By("verifying the helm project namespace has been created by the controller")
				projNs := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: projectNamespace(testProjectName),
					},
				}
				Eventually(Object(projNs), 60*time.Second).Should(ExistAnd(
					HaveLabels(
						labelProjectID, testProjectName,
						labelOperatedByHelmProj, "true",
						labelHelmProj, testProjectName,
					),
					HaveAnnotations(
						labelProjectID, testProjectName,
					),
				))

				By("verifying the helm project operator has created the helm api configmap")
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cfgName,
						Namespace: projectNamespace(testProjectName),
					},
				}
				Eventually(Object(configMap)).Should(Exist())
			})
		})

		When("We create a ProjectHelmChart", func() {
			It("should create the project-helm-chart object", func() {
				projH := v1alpha1.ProjectHelmChart{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testPHCName,
						Namespace: projectNamespace(testProjectName),
					},
					Spec: v1alpha1.ProjectHelmChartSpec{
						HelmAPIVersion: "dummy.cattle.io/v1alpha1",
						Values: v1alpha1.GenericMap{
							"data": map[string]interface{}{
								"hello": "e2e-ci",
							},
						},
					},
				}
				Expect(k8sClient.Create(testCtx, &projH)).To(Succeed())
			})

			It("should create the associated CRs with this project helm charts", func() {
				By("verifying the k3s-io helm-controller has created the helm chart")
				helmchart := &k3shelmv1.HelmChart{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s", testPHCName, releaseName),
						Namespace: chartNs,
					},
				}
				Eventually(Object(helmchart), time.Second*15, time.Millisecond*50).Should(Exist())

				By("verifying the helm locker has created the associated helm release")
				helmchartRelease := &lockerv1alpha1.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s", testPHCName, releaseName),
						Namespace: chartNs,
					},
				}
				Eventually(Object(helmchartRelease), time.Second*15, time.Millisecond*50).Should(Exist())
			})

			It("should create the job which deploys the helm chart", func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("helm-install-%s-%s", testPHCName, releaseName),
						Namespace: chartNs,
					},
				}
				Eventually(Object(job)).Should(Exist())
				// TODO this works, but would be better to mirror the condition in kubectl wait --for=complete
				Eventually(func() error {
					retJob, err := Object(job)()
					if err != nil {
						return err
					}
					if retJob.Status.Succeeded < 1 {
						return fmt.Errorf("job has not yet succeeded")
					}
					return nil
				}).Should(Succeed())
			})

			When("We delete a project helm chart", func() {
				It("should delete the project helm chart CR", func() {
					projH := &v1alpha1.ProjectHelmChart{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testPHCName,
							Namespace: projectNamespace(testProjectName),
						},
						Spec: v1alpha1.ProjectHelmChartSpec{
							HelmAPIVersion: "dummy.cattle.io/v1alpha1",
							Values: v1alpha1.GenericMap{
								"data": map[string]interface{}{
									"hello": "e2e-ci",
								},
							},
						},
					}
					Expect(k8sClient.Delete(testCtx, projH)).To(Succeed())
				})
				//FIXME: this spec could be flaky
				It("should have created the matching delete job", func() {
					deleteJob := &batchv1.Job{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("helm-delete-%s-%s", testPHCName, releaseName),
							Namespace: chartNs,
						},
					}
					Eventually(Object(deleteJob)).Should(Exist())

					Eventually(func() error {
						retJob, err := Object(deleteJob)()
						if err != nil {
							return err
						}
						if retJob.Status.Succeeded < 1 {
							return fmt.Errorf("delete job has not yet succeeded")
						}
						return nil
					}).Should(Succeed())
				})

				It("should make sure that resources that should be absent are absent", func() {
					By("verifying the project helm chart has been deleted")
					projH := &v1alpha1.ProjectHelmChart{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testPHCName,
							Namespace: projectNamespace(testProjectName),
						},
					}
					Consistently(Object(projH)).ShouldNot(Exist())

					By("verifying the helm chart CR has been deleted")
					helmchart := &k3shelmv1.HelmChart{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s", testPHCName, releaseName),
							Namespace: chartNs,
						},
					}
					Consistently(Object(helmchart)).Should(Not(Exist()))

					By("verifying the helm locker release CR has been deleted")
					helmchartRelease := &lockerv1alpha1.HelmRelease{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s", testPHCName, releaseName),
							Namespace: chartNs,
						},
					}
					Consistently(Object(helmchartRelease)).Should(Not(Exist()))
				})
			})
		})
	})
})
