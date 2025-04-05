package common

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// HardeningOptions are options that can be provided to override the default hardening resources applied to all namespaces
// created by this Project Operator. To disable this, specify DisableHardening in the RuntimeOptions.
type HardeningOptions struct {
	// ServiceAccount represents the overrides to be supplied to the default service account patched by the hardening controller
	ServiceAccount *DefaultServiceAccountOptions `yaml:"serviceAccountSpec"`
	// NetworkPolicy represents the overrides to be supplied to the generated NetworkPolicy created by the hardening controller
	NetworkPolicy *DefaultNetworkPolicyOptions `yaml:"networkPolicySpec"`
}

// DefaultServiceAccountOptions represents the overrides to be supplied to the default Service Account's fields
// Note: the values of these fields is identical to what is defined on the corev1.ServiceAccount object
type DefaultServiceAccountOptions struct {
	Secrets                      []corev1.ObjectReference      `yaml:"secrets,omitempty"`
	ImagePullSecrets             []corev1.LocalObjectReference `yaml:"imagePullSecrets,omitempty"`
	AutomountServiceAccountToken *bool                         `yaml:"automountServiceAccountToken,omitEmpty"`
}

// DefaultNetworkPolicyOptions is the NetworkPolicySpec specified for generated NetworkPolicy created by the hardening controller
type DefaultNetworkPolicyOptions networkingv1.NetworkPolicySpec

// LoadHardeningOptionsFromFile unmarshalls the struct found at the file to YAML and reads it into memory
func LoadHardeningOptionsFromFile(path string) (HardeningOptions, error) {
	var hardeningOptions HardeningOptions
	wd, err := os.Getwd()
	if err != nil {
		return HardeningOptions{}, err
	}
	abspath := filepath.Join(wd, path)
	_, err = os.Stat(abspath)
	if err != nil {
		if os.IsNotExist(err) {
			// we just assume the default is used
			err = nil
		}
		return HardeningOptions{}, err
	}
	hardeningOptionsBytes, err := os.ReadFile(abspath)
	if err != nil {
		return hardeningOptions, err
	}
	return hardeningOptions, yaml.UnmarshalStrict(hardeningOptionsBytes, &hardeningOptions)
}
