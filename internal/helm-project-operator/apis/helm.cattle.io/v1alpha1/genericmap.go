package v1alpha1

import (
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:validation:EmbeddedResource

// GenericMap is a wrapper on arbitrary JSON / YAML resources
type GenericMap map[string]interface{}

func (in *GenericMap) DeepCopy() *GenericMap {
	if in == nil {
		return nil
	}
	out := new(GenericMap)
	*out = runtime.DeepCopyJSON(*in)
	return out
}

func (in *GenericMap) ToYAML() ([]byte, error) {
	if in == nil {
		return []byte{}, nil
	}
	return yaml.Marshal(in)
}
