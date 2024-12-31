package release

import (
	"fmt"

	v1alpha1 "github.com/rancher/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	corev1 "k8s.io/api/core/v1"
)

const (
	// HelmReleaseSecretType is the type of a secret that is considered a Helm Release secret
	HelmReleaseSecretType = "helm.sh/release.v1"
)

func releaseKeyToString(key relatedresource.Key) string {
	return fmt.Sprintf("%s/%s", key.Namespace, key.Name)
}

func releaseKeyFromRelease(release *v1alpha1.HelmRelease) relatedresource.Key {
	return relatedresource.Key{
		Namespace: release.Spec.Release.Namespace,
		Name:      release.Spec.Release.Name,
	}
}

func releaseKeyFromSecret(secret *corev1.Secret) *relatedresource.Key {
	if !isHelmReleaseSecret(secret) {
		return nil
	}
	releaseNameFromLabel, ok := secret.GetLabels()["name"]
	if !ok {
		return nil
	}
	return &relatedresource.Key{
		Namespace: secret.GetNamespace(),
		Name:      releaseNameFromLabel,
	}
}

func isHelmReleaseSecret(secret *corev1.Secret) bool {
	return secret.Type == HelmReleaseSecretType
}
