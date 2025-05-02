package integration

import (
	"slices"

	"github.com/google/uuid"
	v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/namespace"
	"github.com/rancher/prometheus-federator/internal/test"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const projectIdLabel = "field.cattle.io/projectId"
const overrideProjectLabel = "x.y.z/projectId"

const image = "rancher/klipper-helm:v0.9.4-build20250113"

func createNsFull(ns *corev1.Namespace) {
	ti := test.GetTestInterface()
	testSetupUid := uuid.New().String()
	ti.ObjectTracker().ObjectTracker(testSetupUid).Add(ns)
	Expect(ti.K8sClient().Create(
		ti.Context(),
		ns,
	)).To(Succeed())

	DeferCleanup(func() {
		ti.ObjectTracker().ObjectTracker(testSetupUid).DeleteAll()
	})
}

func createNs(name string) {
	ti := test.GetTestInterface()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	testSetupUid := uuid.New().String()
	ti.ObjectTracker().ObjectTracker(testSetupUid).Add(ns)
	Expect(ti.K8sClient().Create(
		ti.Context(),
		ns,
	)).To(Succeed())

	DeferCleanup(func() {
		ti.ObjectTracker().ObjectTracker(testSetupUid).DeleteAll()
	})
}

type embedProjectGetter struct {
	projRegistration []string
	system           []string
	// non-mapped target namespaces
	// could probably extend this abstraction to
	// include some sort of mapping of projectHelmChart to target namespaces
	targetNamespaces []string
}

func (e *embedProjectGetter) IsProjectRegistrationNamespace(namespace *corev1.Namespace) bool {
	if namespace == nil {
		return false
	}
	return slices.Contains(e.projRegistration, namespace.Name)
}

func (e *embedProjectGetter) IsSystemNamespace(namespace *corev1.Namespace) bool {
	if namespace == nil {
		return false
	}
	return slices.Contains(e.system, namespace.Name)
}

// GetTargetProjectNamespaces returns the list of namespaces that should be targeted for a given ProjectHelmChart
// Any namespace returned by this should not be a project registration namespace or a system namespace
func (e *embedProjectGetter) GetTargetProjectNamespaces(projectHelmChart *v1alpha1.ProjectHelmChart) ([]string, error) {
	if projectHelmChart == nil {
		return []string{}, nil
	}
	return e.targetNamespaces, nil
}

var _ namespace.ProjectGetter = (*embedProjectGetter)(nil)

func projectGetter(
	projRegistration []string,
	system []string,
	targetNamespaces []string,
) namespace.ProjectGetter {

	return &embedProjectGetter{
		projRegistration: projRegistration,
		system:           system,
		targetNamespaces: targetNamespaces,
	}
}
