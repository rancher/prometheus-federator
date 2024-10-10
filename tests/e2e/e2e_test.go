package e2e_test

import (
	"errors"
	"fmt"
	"os/exec"

	. "github.com/kralicky/kmatch"
	"github.com/rancher/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func createIfNotExist(obj client.Object) {
	err := k8sClient.Create(testCtx, obj)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Fail(fmt.Sprintf("Failed to create object %s", err))
	}
}

const (
	objectSetHash = "objectset.rio.cattle.io/hash"

	objectSetApplied  = "objectset.rio.cattle.io/applied"
	objsetSetId       = "objectset.rio.cattle.io/id"
	objectSetOnwerGVK = "objectset.rio.cattle.io/owner-gvk"
	ownerName         = "objectset.rio.cattle.io/owner-name"
	ownerNamespace    = "objectset.rio.cattle.io/owner-namespace"
)

const (
	exampleReleaseName = "foochart"
	exampleReleaseNs   = "foo"
)

var _ = Describe("E2E helm locker operator tests", Ordered, Label("integration"), func() {
	When("we use the helm locker operator", func() {
		Specify("Expect to find prerequisited CRDs in test cluster", func() {
			// loosely checks that the embedded helm controller is installed
			gvk := schema.GroupVersionKind{
				Group:   "helm.cattle.io",
				Version: "v1",
				Kind:    "HelmChart",
			}

			Eventually(GVK(gvk)).Should(Exist())
		})

		It("should run the helm project operator", func() {
			// TODO : setup helm controller here, once we fix the dependency mess
			ns := "cattle-helm-system"
			err := k8sClient.Create(testCtx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			})
			if err != nil && !apierrors.IsAlreadyExists(err) {
				Fail(fmt.Sprintf("Failed to create namespace %s", err))
			}
		})

		It("Should have applied the helmrelease CRD", func() {
			helmRelease := schema.GroupVersionKind{
				Group:   "helm.cattle.io",
				Version: "v1alpha1",
				Kind:    "HelmRelease",
			}

			Eventually(GVK(helmRelease)).Should(Exist())
		})

		It("should install an example helm chart", func() {
			cmd := exec.CommandContext(
				testCtx,
				"helm",
				"upgrade",
				"--install",
				"-n",
				exampleReleaseNs,
				"--create-namespace",
				exampleReleaseName,
				"../examples/foo-chart",
				"--set",
				"contents=\"abc\"",
			)
			err := cmd.Start()
			Expect(err).NotTo(HaveOccurred(), "Failed to run helm command")
			cmd.Stdout = GinkgoWriter
			cmd.Stderr = GinkgoWriter
			err = cmd.Wait()
			Expect(err).NotTo(HaveOccurred(), "helm upgrade command had a non-zero exit code")

			By("verifying the resource managed by the example chart exists")
			cfg := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-configmap",
					Namespace: exampleReleaseNs,
				},
			}
			Eventually(Object(cfg)).Should(ExistAnd(
				HaveLabels(
					"app.kubernetes.io/managed-by",
					"Helm",
				),
				HaveAnnotations(
					"meta.helm.sh/release-name",
					exampleReleaseName,
					"meta.helm.sh/release-namespace",
					exampleReleaseNs,
				),
				HaveData(
					"contents", "abc",
				),
			))
		})

		When("we create a helm release", func() {
			It("should create a helm release", func() {
				release := &v1alpha1.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-release",
						Namespace: "cattle-helm-system",
					},
					Spec: v1alpha1.HelmReleaseSpec{
						Release: v1alpha1.ReleaseKey{
							Name:      exampleReleaseName,
							Namespace: exampleReleaseNs,
						},
					},
				}
				createIfNotExist(release)

				By("Verifing it has the appropriate annotations and finalizers")
				Eventually(Object(release)).Should(
					ExistAnd(
						HaveAnnotations(
							"helmreleases.cattle.io/managed-by", "helm-locker",
						),
						HaveFinalizers("wrangler.cattle.io/on-helm-release-remove"),
					),
				)

				By("Verifying the helm-locker is consistently in the deployed state", func() {
					extractState := func() string {
						retRelease, err := Object(release)()
						if err != nil {
							return v1alpha1.UnknownState
						}
						return retRelease.Status.State
					}
					Consistently(extractState).Should(Equal(v1alpha1.DeployedState))
				})
			})

			Specify("We should not be able to edit or delete resources managed by the helm-chart", func() {
				By("verifying the config map has the correct objectset annotations and labels")
				origHash := ""
				origApplied := ""
				cfg := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-configmap",
						Namespace: "foo",
					},
					Data: map[string]string{
						"contents": "Hello, World! Updated",
					},
				}
				Eventually(func() error {
					errs := []error{}

					retCfg, err := Object(cfg)()
					if err != nil {
						return err
					}

					if val, ok := retCfg.Labels[objectSetHash]; !ok {
						errs = append(errs, errors.New("objectset hash label not found"))
					} else {
						origHash = val
					}

					if val, ok := retCfg.Annotations[objectSetApplied]; !ok {
						errs = append(errs, errors.New("objectset hash not found or incorrect"))
					} else {
						origApplied = val
					}

					if val, ok := retCfg.Annotations[objsetSetId]; !ok || val != "object-set-applier" {
						errs = append(errs, fmt.Errorf("objectset id not found or incorrect: '%s'", val))
					}
					if val, ok := retCfg.Annotations[objectSetOnwerGVK]; !ok || val != "internal.cattle.io/v1alpha1, Kind=objectSetState" {
						errs = append(errs, fmt.Errorf("objectset owner gvk not found or incorrect '%s'", val))
					}
					return errors.Join(errs...)
				}).Should(Succeed())

				Expect(origHash).NotTo(BeEmpty(), "helm locker should manage the objectset hash")
				Expect(origApplied).NotTo(BeEmpty(), "helm locker should manage the objectset applied annotation")

				By("trying to update the helm locked resource")
				Expect(k8sClient.Update(testCtx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-configmap",
						Namespace: "foo",
					},
					Data: map[string]string{
						"contents": "Hello, World! Updated",
					},
				})).To(Succeed())

				By("verifying the update was not applied")
				Eventually(Object(cfg)).Should(ExistAnd(
					HaveData(
						"contents", "abc",
					),
					HaveAnnotations(
						ownerName, exampleReleaseName,
						ownerNamespace, exampleReleaseNs,
					),
				))
				Eventually(func() error {
					retCfg, err := Object(cfg)()
					if err != nil {
						return err
					}
					if val, ok := retCfg.Labels[objectSetHash]; !ok || val != origHash {
						return fmt.Errorf("objectset hash label does not match the original one : '%s' vs '%s'", origHash, val)
					}
					if val, ok := retCfg.Annotations[objectSetApplied]; !ok || val != origApplied {
						return fmt.Errorf("objectset applied annotation does not match the original one : '%s' vs '%s'", origApplied, val)
					}
					return nil
				}).Should(Succeed())

				Consistently(Object(cfg)).Should(ExistAnd(
					HaveData(
						"contents", "abc",
					),
					HaveAnnotations(
						ownerName, exampleReleaseName,
						ownerNamespace, exampleReleaseNs,
					),
				))
			})

			Specify("We should only be able to update resources managed by the helm chart through helm", func() {
				cfg := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-configmap",
						Namespace: exampleReleaseNs,
					},
				}
				By("extracting the objectset hash before the helm update")
				origHash := ""
				origApplied := ""
				Eventually(func() error {
					retCfg, err := Object(cfg)()
					if err != nil {
						return err
					}
					if val, ok := retCfg.Labels[objectSetHash]; ok {
						origHash = val
					}
					if val, ok := retCfg.Annotations[objectSetApplied]; ok {
						origApplied = val
					}
					return nil
				}).Should(Succeed())
				Expect(origHash).NotTo(BeEmpty(), "helm locker should be managing the object set hash")
				Expect(origApplied).NotTo(BeEmpty(), "helm locker should be managing the object set hash")
				Eventually(func() error {
					return nil
				}).Should(Succeed())

				By("upgrading the helm resource using helm")
				cmd := exec.CommandContext(
					testCtx,
					"helm",
					"upgrade",
					"--install",
					"-n",
					exampleReleaseNs,
					"--create-namespace",
					exampleReleaseName,
					"../examples/foo-chart",
					"--set",
					"contents=\"Updated!\"",
				)
				err := cmd.Start()
				Expect(err).NotTo(HaveOccurred(), "Failed to run helm command")

				err = cmd.Wait()
				Expect(err).NotTo(HaveOccurred(), "helm install command had a non-zero exit code")

				By("verifying the resource managed by the example chart exists")
				Eventually(Object(cfg)).Should(ExistAnd(
					HaveLabels(
						"app.kubernetes.io/managed-by",
						"Helm",
					),
					HaveAnnotations(
						"meta.helm.sh/release-name",
						exampleReleaseName,
						"meta.helm.sh/release-namespace",
						exampleReleaseNs,
					),
					HaveData(
						"contents", "Updated!",
					),
				))

				Consistently(Object(cfg)).Should(ExistAnd(
					HaveData(
						"contents", "Updated!",
					),
				))

				By("extracting the objectset hash after the helm update")
				newHash := ""
				newApplied := ""
				Eventually(func() error {
					retCfg, err := Object(cfg)()
					if err != nil {
						return err
					}
					if val, ok := retCfg.Labels[objectSetHash]; ok {
						newHash = val
					}
					if val, ok := retCfg.Annotations[objectSetApplied]; ok {
						newApplied = val
					}
					return nil
				}).Should(Succeed())
				Expect(newHash).NotTo(BeEmpty(), "helm locker should be managing the object set hash")
				Expect(newApplied).NotTo(BeEmpty(), "helm locker should be managing the object set hash")
				Expect(newHash).To(Equal(origHash), "objectset hash should not have changed after helm update, since no new resource keys are tracked")
				Expect(newApplied).NotTo(
					Equal(origApplied),
					"objectset applied annotation should have changed after helm update",
				)
			})
		})

		When("we delete the helm release", func() {
			It("should remove the helm release", func() {
				release := &v1alpha1.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-release",
						Namespace: "cattle-helm-system",
					},
				}
				err := k8sClient.Delete(testCtx, release)
				Expect(err).ToNot(HaveOccurred())

				By("Verifing it has the appropriate annotations and finalizers")
				Eventually(Object(release)).Should(Not(Exist()))
			})

			Specify("we should be able to edit and delete resources managed by the helm-chart", func() {
				Expect(k8sClient.Update(testCtx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-configmap",
						Namespace: exampleReleaseNs,
					},
					Data: map[string]string{
						"contents": "Hello, World! Updated",
					},
				})).To(Succeed())

				By("verifying the update was applied")
				cfg := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-configmap",
						Namespace: exampleReleaseNs,
					},
				}
				Eventually(Object(cfg)).Should(ExistAnd(
					HaveData(
						"contents", "Hello, World! Updated",
					),
				))

				Consistently(Object(cfg)).Should(ExistAnd(
					HaveData(
						"contents", "Hello, World! Updated",
					),
				))
			})
		})
	})
})
