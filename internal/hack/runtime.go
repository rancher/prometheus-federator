package hack

import (
	"context"
	"slices"
	"strings"

	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/name"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/sirupsen/logrus"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CRDManagementConfig struct {
	options common.RuntimeOptions
}

func CRDName(crd crd.CRD) string {
	return crd.GVK.GroupVersion().WithKind(strings.ToLower(name.GuessPluralName(crd.GVK.Kind))).GroupKind().String()
}

func FilterMissingCRDs(client apiextensionv1.CustomResourceDefinitionInterface, expectedCRDs []crd.CRD) ([]crd.CRD, error) {
	ret := []crd.CRD{}
	for _, currentCRD := range expectedCRDs {
		crdName := CRDName(currentCRD)
		// try to get the given CRD just to check for error, verifying if it exists
		foundCRD, err := client.Get(context.TODO(), crdName, metav1.GetOptions{})
		if err == nil {
			storedVersions := foundCRD.Status.StoredVersions
			logrus.Debugf(
				"Found `%s` at versions `%s`, expecting version `%s`",
				crdName,
				strings.Join(storedVersions, ","),
				currentCRD.GVK.Version,
			)
			if slices.Contains(storedVersions, currentCRD.GVK.Version) {
				logrus.Debugf("Installing `%s` will be skipped; a suitible version exists on the cluster", crdName)
				continue
			}
			logrus.Debugf("No suitable version `%s` found for `%s`, queuing it for install", currentCRD.GVK.Version, crdName)
			ret = append(ret, currentCRD)
		} else if !apierrors.IsNotFound(err) {
			return nil, err
		} else {
			logrus.Debugf("Did not find `%s` on the cluster, it will be installed", crdName)
			ret = append(ret, currentCRD)
		}
	}
	return ret, nil
}
