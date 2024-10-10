package release

import (
	"context"
	"fmt"

	v1alpha1 "github.com/rancher/helm-project-operator/pkg/helm-locker/apis/helm.cattle.io/v1alpha1"
	helmcontroller "github.com/rancher/helm-project-operator/pkg/helm-locker/generated/controllers/helm.cattle.io/v1alpha1"
	"github.com/rancher/helm-project-operator/pkg/helm-locker/objectset"
	"github.com/rancher/helm-project-operator/pkg/helm-locker/objectset/parser"
	"github.com/rancher/helm-project-operator/pkg/helm-locker/releases"
	"github.com/rancher/helm-project-operator/pkg/remove"
	"github.com/rancher/lasso/pkg/controller"
	corecontroller "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

const (
	// HelmReleaseByReleaseKey is the key used to get HelmRelease objects by the namespace/name of the underlying Helm Release it points to
	HelmReleaseByReleaseKey = "helm.cattle.io/helm-release-by-release-key"

	// ManagedBy is an annotation attached to HelmRelease objects that indicates that they are managed by this operator
	ManagedBy = "helmreleases.cattle.io/managed-by"
)

type handler struct {
	systemNamespace string
	managedBy       string

	helmReleases     helmcontroller.HelmReleaseController
	helmReleaseCache helmcontroller.HelmReleaseCache
	secrets          corecontroller.SecretController
	secretCache      corecontroller.SecretCache

	releases releases.HelmReleaseGetter

	lockableObjectSetRegister objectset.LockableRegister
	recorder                  record.EventRecorder
}

func Register(
	ctx context.Context,
	systemNamespace, managedBy string,
	helmReleases helmcontroller.HelmReleaseController,
	helmReleaseCache helmcontroller.HelmReleaseCache,
	secrets corecontroller.SecretController,
	secretCache corecontroller.SecretCache,
	k8s kubernetes.Interface,
	lockableObjectSetRegister objectset.LockableRegister,
	lockableObjectSetHandler *controller.SharedHandler,
	recorder record.EventRecorder,
) {

	h := &handler{
		systemNamespace: systemNamespace,
		managedBy:       managedBy,

		helmReleases:     helmReleases,
		helmReleaseCache: helmReleaseCache,
		secrets:          secrets,
		secretCache:      secretCache,

		releases: releases.NewHelmReleaseGetter(k8s),

		lockableObjectSetRegister: lockableObjectSetRegister,
		recorder:                  recorder,
	}

	lockableObjectSetHandler.Register(ctx, "on-objectset-change", controller.SharedControllerHandlerFunc(h.OnObjectSetChange))

	helmReleaseCache.AddIndexer(HelmReleaseByReleaseKey, helmReleaseToReleaseKey)

	relatedresource.Watch(ctx, "on-helm-secret-change", h.resolveHelmRelease, helmReleases, secrets)

	helmReleases.OnChange(ctx, "apply-lock-on-release", h.OnHelmRelease)

	remove.RegisterScopedOnRemoveHandler(ctx, helmReleases, "on-helm-release-remove",
		func(_ string, obj runtime.Object) (bool, error) {
			if obj == nil {
				return false, nil
			}
			helmRelease, ok := obj.(*v1alpha1.HelmRelease)
			if !ok {
				return false, nil
			}
			return h.shouldManage(helmRelease)
		},
		helmcontroller.FromHelmReleaseHandlerToHandler(h.OnHelmReleaseRemove),
	)
}

func (h *handler) OnObjectSetChange(setID string, obj runtime.Object) (runtime.Object, error) {
	helmReleases, err := h.helmReleaseCache.GetByIndex(HelmReleaseByReleaseKey, setID)
	if err != nil {
		return nil, fmt.Errorf("unable to find HelmReleases for objectset %s to trigger event", setID)
	}
	for _, helmRelease := range helmReleases {
		if helmRelease == nil {
			continue
		}
		if obj != nil {
			h.recorder.Eventf(helmRelease, corev1.EventTypeNormal, "Locked", "Applied ObjectSet %s tied to HelmRelease %s/%s to lock into place", setID, helmRelease.Namespace, helmRelease.Name)
		} else {
			h.recorder.Eventf(helmRelease, corev1.EventTypeNormal, "Untracked", "ObjectSet %s tied to HelmRelease %s/%s is not tracked", setID, helmRelease.Namespace, helmRelease.Name)
		}
	}
	return nil, nil
}

func helmReleaseToReleaseKey(helmRelease *v1alpha1.HelmRelease) ([]string, error) {
	releaseKey := releaseKeyFromRelease(helmRelease)
	return []string{releaseKeyToString(releaseKey)}, nil
}

func (h *handler) resolveHelmRelease(_ /* secretNamespace */, _ /* secretName */ string, obj runtime.Object) ([]relatedresource.Key, error) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil, nil
	}
	releaseKey := releaseKeyFromSecret(secret)
	if releaseKey == nil {
		// No release found matching this secret
		return nil, nil
	}
	helmReleases, err := h.helmReleaseCache.GetByIndex(HelmReleaseByReleaseKey, releaseKeyToString(*releaseKey))
	if err != nil {
		return nil, err
	}

	keys := make([]relatedresource.Key, len(helmReleases))
	for i, helmRelease := range helmReleases {
		keys[i] = relatedresource.Key{
			Name:      helmRelease.Name,
			Namespace: helmRelease.Namespace,
		}
	}

	return keys, nil
}

// shouldManage determines if this HelmRelease should be handled by this operator
func (h *handler) shouldManage(helmRelease *v1alpha1.HelmRelease) (bool, error) {
	if helmRelease == nil {
		return false, nil
	}
	if helmRelease.Namespace != h.systemNamespace {
		return false, nil
	}
	if helmRelease.Annotations != nil {
		managedBy, ok := helmRelease.Annotations[ManagedBy]
		if ok {
			// if the label exists, only handle this if the managedBy label matches that of this controller
			return managedBy == h.managedBy, nil
		}
	}
	// The managedBy label does not exist, so we trigger claiming the HelmRelease
	// We then return false since this update will automatically retrigger an OnChange operation
	helmReleaseCopy := helmRelease.DeepCopy()
	if helmReleaseCopy.Annotations == nil {
		helmReleaseCopy.SetAnnotations(map[string]string{
			ManagedBy: h.managedBy,
		})
	} else {
		helmReleaseCopy.Annotations[ManagedBy] = h.managedBy
	}
	_, err := h.helmReleases.Update(helmReleaseCopy)
	return false, err
}

func (h *handler) OnHelmReleaseRemove(_ string, helmRelease *v1alpha1.HelmRelease) (*v1alpha1.HelmRelease, error) {
	if helmRelease == nil {
		return nil, nil
	}
	if helmRelease.Status.State == v1alpha1.SecretNotFoundState || helmRelease.Status.State == v1alpha1.UninstalledState {
		// HelmRelease was not tracking any underlying objectSet
		return helmRelease, nil
	}
	// HelmRelease CRs are only pointers to Helm releases... if the HelmRelease CR is removed, we should do nothing, but should warn the user
	// that we are leaving behind resources in the cluster
	logrus.Warnf("HelmRelease %s/%s was removed, resources tied to Helm release may need to be manually deleted", helmRelease.Namespace, helmRelease.Name)
	logrus.Warnf("To delete the contents of a Helm release automatically, delete the Helm release secret before deleting the HelmRelease.")
	releaseKey := releaseKeyFromRelease(helmRelease)
	h.lockableObjectSetRegister.Delete(releaseKey, false) // remove the objectset, but don't purge the underlying resources
	return helmRelease, nil
}

func (h *handler) OnHelmRelease(_ string, helmRelease *v1alpha1.HelmRelease) (*v1alpha1.HelmRelease, error) {
	if shouldManage, err := h.shouldManage(helmRelease); err != nil {
		return helmRelease, err
	} else if !shouldManage {
		return helmRelease, nil
	}
	if helmRelease.DeletionTimestamp != nil {
		return helmRelease, nil
	}
	releaseKey := releaseKeyFromRelease(helmRelease)
	latestRelease, err := h.releases.Last(releaseKey.Namespace, releaseKey.Name)
	if err != nil {
		if err == driver.ErrReleaseNotFound {
			logrus.Warnf("waiting for release %s/%s to be found to reconcile HelmRelease %s, deleting any orphaned resources", releaseKey.Namespace, releaseKey.Name, helmRelease.GetName())
			h.lockableObjectSetRegister.Delete(releaseKey, true) // remove the objectset and purge any untracked resources
			helmRelease.Status.Version = 0
			helmRelease.Status.Description = "Could not find Helm Release Secret"
			helmRelease.Status.State = v1alpha1.SecretNotFoundState
			helmRelease.Status.Notes = ""
			return h.helmReleases.UpdateStatus(helmRelease)
		}
		return helmRelease, fmt.Errorf("unable to find latest Helm Release Secret tied to Helm Release %s: %s", helmRelease.GetName(), err)
	}
	logrus.Infof("loading latest release version %d of HelmRelease %s", latestRelease.Version, helmRelease.GetName())
	releaseInfo := newReleaseInfo(latestRelease)
	helmRelease, err = h.helmReleases.UpdateStatus(releaseInfo.GetUpdatedStatus(helmRelease))
	if err != nil {
		return helmRelease, fmt.Errorf("unable to update status of HelmRelease %s: %s", helmRelease.GetName(), err)
	}
	if !releaseInfo.Locked() {
		// TODO: add status
		logrus.Infof("detected HelmRelease %s is not deployed or transitioning (state is %s), unlocking release", helmRelease.GetName(), releaseInfo.State)
		h.lockableObjectSetRegister.Unlock(releaseKey)
		h.recorder.Eventf(helmRelease, corev1.EventTypeNormal, "Transitioning", "Unlocked HelmRelease %s/%s to allow changes while Helm operation is being executed", helmRelease.Namespace, helmRelease.Name)
		return helmRelease, nil
	}
	manifestOS, err := parser.Parse(releaseInfo.Manifest)
	if err != nil {
		// TODO: add status
		return helmRelease, fmt.Errorf("unable to parse objectset from manifest for HelmRelease %s: %s", helmRelease.GetName(), err)
	}
	logrus.Infof("detected HelmRelease %s is deployed, locking release %s with %d objects", helmRelease.GetName(), releaseKey, len(manifestOS.All()))
	locked := true
	h.lockableObjectSetRegister.Set(releaseKey, manifestOS, &locked)
	return helmRelease, nil
}
