package namespace

import (
	"fmt"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// initSystemNamespaces initializes all System Namespaces on the Tracker
func (h *handler) initSystemNamespaces(systemNamespaceList []string, systemNamespaceTracker Tracker) {
	for _, namespace := range systemNamespaceList {
		systemNamespaceTracker.Set(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
	}
}

// initProjectRegistrationNamespaces initializes all Project Registration Namespaces on the Tracker
// It also automatically triggers the creation of the Project Registration Namespaces if necessary
func (h *handler) initProjectRegistrationNamespaces() error {
	namespaceList, err := h.namespaces.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("unable to list namespaces to enqueue all Helm charts: %s", err)
	}
	if namespaceList != nil {
		logrus.Infof("Identifying and registering projectRegistrationNamespaces...")
		// trigger the OnChange events for all namespaces before returning on a register
		//
		// this ensures that registration will create projectRegistrationNamespaces and
		// have isProjectRegistration and isSystemNamespace up to sync before it provides
		// the ProjectGetter interface to other controllers that need it.
		//
		// Q: Why don't we use Enqueue here?
		//
		// Enqueue will add it to the workqueue but there's no guarantee the namespace's processing
		// will happen before this function exits, which is what we need to guarantee here.
		// As a result, we explicitly call OnChange here to force the apply to happen and wait for it to finish
		for _, ns := range namespaceList.Items {
			_, err := h.OnMultiNamespaceChange(ns.Name, &ns)
			if err != nil {
				// encountered some error, just fail to start
				// Possible TODO: Perhaps we should add a backoff retry here?
				return fmt.Errorf("unable to initialize projectRegistrationNamespaces before starting other handlers that utilize ProjectGetter: %s", err)
			}
		}
	}
	return nil
}
