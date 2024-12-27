package objectset

import (
	"fmt"

	"github.com/rancher/helm-locker/pkg/gvk"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/apply"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

type handler struct {
	apply     apply.Apply
	gvkLister gvk.Lister
	locker    Locker

	// allows us to add hooks into triggering certain actions on reconciles, e.g. launching events
	sharedHandler *controller.SharedHandler
}

// configureApply configures the apply object for the provided setID and objectSetState
func (h *handler) configureApply(setID string, oss *objectSetState) apply.Apply {
	apply := h.apply.
		WithSetID("object-set-applier").
		WithOwnerKey(setID, internalGroupVersion.WithKind("objectSetState"))

	if oss != nil && oss.ObjectSet != nil {
		apply = apply.WithGVK(oss.ObjectSet.GVKs()...)
	} else {
		// if we cannot infer the GVK from the provided object set, include all GVKs in the cache types
		gvks, err := h.gvkLister.List()
		if err != nil {
			logrus.Errorf("unable to list GVKs to apply deletes on objects, objectset %s may require manual cleanup: %s", setID, err)
		} else {
			apply = apply.WithGVK(gvks...)
		}
	}

	return apply
}

// OnChange reconciles the resources tracked by an objectSetState
func (h *handler) OnChange(setID string, obj runtime.Object) error {
	logrus.Debugf("on change: %s", setID)

	if obj == nil {
		// nothing to do
		return nil
	}
	oss, ok := obj.(*objectSetState)
	if !ok {
		return fmt.Errorf("expected object of type objectSetState, found %t", obj)
	}

	if oss.DeletionTimestamp != nil {
		return nil
	}

	key := relatedresource.FromString(setID)
	h.locker.Unlock(key) // ensure that apply does not trigger locking again

	if !oss.Locked {
		// nothing to do
		return nil
	}
	// Run the apply
	defer h.locker.Lock(key)

	logrus.Debugf("running apply for %s...", setID)
	if err := h.configureApply(setID, oss).Apply(oss.ObjectSet); err != nil {
		return fmt.Errorf("failed to apply objectset for %s: %s", setID, err)
	}

	logrus.Infof("applied %s", setID)

	go h.sharedHandler.OnChange(setID, obj)

	return nil
}

// OnRemove cleans up the resources tracked by an objectSetState
func (h *handler) OnRemove(setID string, purge bool) {
	logrus.Debugf("on delete: %s", setID)

	key := relatedresource.FromString(setID)

	h.locker.Unlock(key)

	if !purge {
		return
	}

	logrus.Debugf("running apply for %s...", setID)
	if err := h.configureApply(setID, nil).ApplyObjects(); err != nil {
		logrus.Errorf("failed to clean up objectset %s: %s", setID, err)
	}

	logrus.Infof("applied %s", setID)

	go h.sharedHandler.OnChange(setID, nil)
}
