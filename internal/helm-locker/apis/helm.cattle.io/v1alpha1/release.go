package v1alpha1

import (
	"github.com/rancher/wrangler/pkg/genericcondition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Helm Release Statuses

	// SecretNotFoundState is the state when a Helm release secret has not been found for this HelmRelease
	SecretNotFoundState = "SecretNotFound"

	// UnknownState is the state when the Helm release secret reports that it does not know the state of the underlying Helm release
	UnknownState = "Unknown"

	// DeployedState is the state where the underlying Helm release has been successfully deployed, indicating Helm Locker should lock the release
	DeployedState = "Deployed"

	// UninstalledState is the state when the underlying Helm release is uninstalled but the Helm release secret has not been deleted
	UninstalledState = "Uninstalled"

	// ErrorState is a state where Helm Locker has encountered an unexpected bug on trying to parse the underlying Helm release
	ErrorState = "Error"

	// FailedState is the state when the underlying Helm release has failed its last Helm operation
	FailedState = "Failed"

	// TransitioningState is the transitionary state when a Helm operation is being performed on the release (install, upgrade, uninstall)
	TransitioningState = "Transitioning"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type HelmRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              HelmReleaseSpec   `json:"spec"`
	Status            HelmReleaseStatus `json:"status"`
}

type HelmReleaseSpec struct {
	Release ReleaseKey `json:"release,omitempty"`
}

type ReleaseKey struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type HelmReleaseStatus struct {
	State       string `json:"state,omitempty"`
	Version     int    `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	Notes       string `json:"notes,omitempty"`

	Conditions []genericcondition.GenericCondition `json:"conditions,omitempty"`
}
