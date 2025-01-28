package parser

import (
	"bytes"

	"github.com/rancher/wrangler/pkg/objectset"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Parse parses the runtime.Objects tracked in a Kubernetes manifest (represented as a string) into an ObjectSet
func Parse(manifest string) (*objectset.ObjectSet, error) {
	var multierr error

	var u unstructured.Unstructured
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(manifest)), 1000)
	os := objectset.NewObjectSet()
	for {
		uCopy := u.DeepCopy()
		err := decoder.Decode(uCopy)
		if err != nil {
			break
		}
		if uCopy.GetAPIVersion() == "" || uCopy.GetKind() == "" {
			// Encountered empty YAML document but successfully decoded, skip
			continue
		}
		os = os.Add(uCopy)
		logrus.Debugf("obj: %s, Kind=%s (%s/%s)", uCopy.GetAPIVersion(), uCopy.GetKind(), uCopy.GetName(), uCopy.GetNamespace())
	}
	return os, multierr
}
