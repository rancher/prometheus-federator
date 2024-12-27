package crds

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rancher/wrangler/v3/pkg/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// WriteFiles writes CRDs and dependent CRDs to the paths specified
//
// Note: It is recommended to write CRDs to the templates directory (or similar) and to write
// CRD dependencies to the crds/ directory since you do not want the uninstall or upgrade of the
// CRD chart to destroy existing dependent CRDs in the cluster as that could break other components
//
// i.e. if you uninstall the HelmChart CRD, it can destroy an RKE2 or K3s cluster that also uses those CRs
// to manage internal Kubernetes component state
func WriteFiles(
	tracker CRDTracker, crdDirpath string) error {
	objs, err := tracker.Objects()
	if err != nil {
		return err
	}
	return writeFiles(crdDirpath, objs)
}

func writeFiles(dirpath string, objs []runtime.Object) error {
	if err := os.MkdirAll(dirpath, 0755); err != nil {
		return err
	}

	objMap := make(map[string][]byte)

	for _, o := range objs {
		data, err := yaml.Export(o)
		if err != nil {
			return err
		}
		metaData, err := meta.Accessor(o)
		if err != nil {
			return err
		}
		key := strings.SplitN(metaData.GetName(), ".", 2)[0]
		objMap[key] = data
	}

	var wg sync.WaitGroup
	wg.Add(len(objMap))
	for key, data := range objMap {
		go func(key string, data []byte) {
			defer wg.Done()
			f, err := os.Create(filepath.Join(dirpath, fmt.Sprintf("%s.yaml", key)))
			if err != nil {
				logrus.Error(err)
			}
			defer f.Close()
			_, err = f.Write(data)
			if err != nil {
				logrus.Error(err)
			}
		}(key, data)
	}
	wg.Wait()

	return nil
}
