package namespace

import (
	"context"

	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/unstructured"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

// addReconcilers registers reconcilers on the apply object that configure how it reconciles changes to specific resources
func (h *handler) addReconcilers(apply apply.Apply, dynamic dynamic.Interface) apply.Apply {
	// force recreate configmaps since configmaps can have errors on updates
	// for example, if a configmap has been modified to have immutable set to true, it will encounter an error
	// another example is if a user tries to switch a key from data to binaryData or vice versa; in this case,
	// the k8s API will throw an error due to trying to move a field across locations
	r := forceRecreator{
		NamespaceableResourceInterface: dynamic.Resource(corev1.SchemeGroupVersion.WithResource("configmaps")),
	}
	apply = apply.WithReconciler(corev1.SchemeGroupVersion.WithKind("ConfigMap"), r.deleteAndReplace)

	logrus.Infof("Adding reconcilers on the apply object %s", apply)
	return apply
}

// forceRecreator is a wrapper on the dynamic.NamespaceableResourceInterface that implements an apply.Reconciler
// that uses the interface to delete and recreate a dynamic object on reconcile
type forceRecreator struct {
	dynamic.NamespaceableResourceInterface

	deleteOptions metav1.DeleteOptions
	createOptions metav1.CreateOptions
}

func (r *forceRecreator) deleteAndReplace(oldObj runtime.Object, newObj runtime.Object) (bool, error) {
	meta, err := meta.Accessor(oldObj)
	if err != nil {
		return false, err
	}
	nsed := r.NamespaceableResourceInterface.Namespace(meta.GetNamespace())
	// convert newObj to unstructured
	uNewObj, err := unstructured.ToUnstructured(newObj)
	if err != nil {
		return false, err
	}
	// perform delete and recreate
	if err := nsed.Delete(context.TODO(), meta.GetName(), r.deleteOptions); err != nil {
		return false, err
	}
	if _, err := nsed.Create(context.TODO(), uNewObj, r.createOptions); err != nil {
		return false, err
	}
	return true, nil
}
