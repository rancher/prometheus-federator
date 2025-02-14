package hack_test

import (
	"fmt"
	"testing"

	helmcontrollercrd "github.com/k3s-io/helm-controller/pkg/crd"
	"github.com/rancher/prometheus-federator/internal/hack"
	lockercrd "github.com/rancher/prometheus-federator/internal/helm-locker/pkg/crd"
	helmprojectcrds "github.com/rancher/prometheus-federator/internal/helm-project-operator/pkg/crd"
	mock "github.com/rancher/prometheus-federator/internal/test/mocks"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestFilterMissingCRDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock.NewMockCustomResourceDefinitionInterface(ctrl)

	crds := lockercrd.Required()
	expectedCrds := helmprojectcrds.Required()
	expectedCrds = append(expectedCrds, helmcontrollercrd.List()...)

	client.EXPECT().Get(
		gomock.Any(),
		"helmreleases.helm.cattle.io",
		gomock.Any(),
	).Return(
		&apiextensionsv1.CustomResourceDefinition{
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1alpha1"},
			},
		},
		nil,
	)

	client.EXPECT().Get(
		gomock.Any(),
		"helmcharts.helm.cattle.io",
		gomock.Any(),
	).Return(nil, apierrors.NewNotFound(schema.GroupResource{}, "blanket not found error"))

	client.EXPECT().Get(
		gomock.Any(),
		"helmchartconfigs.helm.cattle.io",
		gomock.Any(),
	).Return(nil, apierrors.NewNotFound(schema.GroupResource{}, "blanket not found error"))

	client.EXPECT().Get(
		gomock.Any(),
		"projecthelmcharts.helm.cattle.io",
		gomock.Any(),
	).Return(nil, apierrors.NewNotFound(schema.GroupResource{}, "blanket not found error"))

	crdSet, err := hack.FilterMissingCRDs(client, append(crds, expectedCrds...))
	require.NoError(t, err)
	expected := lo.Map(crdSet, func(c crd.CRD, _ int) string {
		return hack.CRDName(c)
	})

	require.Equal(t, expected, []string{"projecthelmcharts.helm.cattle.io", "helmcharts.helm.cattle.io", "helmchartconfigs.helm.cattle.io"})
}

func TestFilterMissingCrdsUpgrade(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock.NewMockCustomResourceDefinitionInterface(ctrl)

	crds := lockercrd.Required()

	client.EXPECT().Get(
		gomock.Any(),
		"helmreleases.helm.cattle.io",
		gomock.Any(),
	).Return(
		&apiextensionsv1.CustomResourceDefinition{
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1beta1", "v1alpha2"},
			},
		},
		nil,
	)

	crdSet, err := hack.FilterMissingCRDs(client, crds)
	require.NoError(t, err)
	ret := lo.Map(crdSet, func(c crd.CRD, _ int) string {
		return hack.CRDName(c)
	})
	require.Equal(t, ret, []string{"helmreleases.helm.cattle.io"})

}

func TestFilterMissingCRDsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock.NewMockCustomResourceDefinitionInterface(ctrl)

	crds := lockercrd.Required()
	client.EXPECT().Get(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(nil, apierrors.NewServerTimeout(schema.GroupResource{}, "get", 60))

	crdSet, err := hack.FilterMissingCRDs(client, crds)
	require.Error(t, err)
	require.Len(t, crdSet, 0)
}

func TestIdentifyKubernetesRuntimeType(t *testing.T) {
	ctrl := gomock.NewController(t)
	node := mock.NewMockNodeInterface(ctrl)

	node.EXPECT().List(gomock.Any(), gomock.Any()).Return(
		&corev1.NodeList{
			Items: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"node.kubernetes.io/instance-type": "k3s",
						},
					},
				},
			},
		},
		nil,
	).AnyTimes()

	rType, err := hack.IdentifyKubernetesRuntimeType(node)
	require.NoError(t, err)
	require.Equal(t, rType, "k3s")

	node2 := mock.NewMockNodeInterface(ctrl)

	node2.EXPECT().List(gomock.Any(), gomock.Any()).Return(
		&corev1.NodeList{
			Items: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"node.kubernetes.io/instance-type": "rke2",
						},
					},
				},
			},
		},
		nil,
	).AnyTimes()

	rType2, err := hack.IdentifyKubernetesRuntimeType(node2)
	require.NoError(t, err)
	require.Equal(t, rType2, "rke2")

	node3 := mock.NewMockNodeInterface(ctrl)

	node3.EXPECT().List(gomock.Any(), gomock.Any()).Return(
		&corev1.NodeList{
			Items: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"node.kubernetes.io/instance-type": "rke2",
						},
					},
				},
				{},
				{},
			},
		},
		nil,
	).AnyTimes()
	rType3, err := hack.IdentifyKubernetesRuntimeType(node3)
	require.NoError(t, err)
	require.Equal(t, rType3, "rke2")
}

func TestIdentifyKubernetesRuntimeTypeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	node := mock.NewMockNodeInterface(ctrl)
	node.EXPECT().List(gomock.Any(), gomock.Any()).Return(
		nil,
		fmt.Errorf("any error"),
	).AnyTimes()

	_, err := hack.IdentifyKubernetesRuntimeType(node)
	require.Error(t, err)

	node2 := mock.NewMockNodeInterface(ctrl)

	node2.EXPECT().List(
		gomock.Any(),
		gomock.Any(),
	).Return(
		&corev1.NodeList{
			Items: []corev1.Node{
				{},
				{},
			},
		},
		nil,
	)

	_, err = hack.IdentifyKubernetesRuntimeType(node2)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot identify k8s runtime type")
}
