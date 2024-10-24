package crd

import (
	"context"
	"fmt"
	"github.com/rancher/wrangler/pkg/name"
	"io"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"

	helmcontrollercrd "github.com/k3s-io/helm-controller/pkg/crd"
	helmlockercrd "github.com/rancher/prometheus-federator/pkg/helm-locker/crd"
	"github.com/rancher/wrangler/pkg/crd"
	"github.com/rancher/wrangler/pkg/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// WriteFiles writes CRDs and dependent CRDs to the paths specified
//
// Note: It is recommended to write CRDs to the templates directory (or similar) and to write
// CRD dependencies to the crds/ directory since you do not want the uninstall or upgrade of the
// CRD chart to destroy existing dependent CRDs in the cluster as that could break other components
//
// i.e. if you uninstall the HelmChart CRD, it can destroy an RKE2 or K3s cluster that also uses those CRs
// to manage internal Kubernetes component state
func WriteFiles(crdDirpath, crdDepDirpath string) error {
	objs, depObjs, err := Objects(false)
	if err != nil {
		return err
	}
	if err := writeFiles(crdDirpath, objs); err != nil {
		return err
	}
	return writeFiles(crdDepDirpath, depObjs)
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
		meta, err := meta.Accessor(o)
		if err != nil {
			return err
		}
		key := strings.SplitN(meta.GetName(), ".", 2)[0]
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

// Print prints CRDs to out and dependent CRDs to depOut
func Print(out io.Writer, depOut io.Writer) {
	objs, depObjs, err := Objects(false)
	if err != nil {
		logrus.Fatalf("%s", err)
	}
	if err := printCrd(out, objs); err != nil {
		logrus.Fatalf("%s", err)
	}
	if err := printCrd(depOut, depObjs); err != nil {
		logrus.Fatalf("%s", err)
	}
}

func printCrd(out io.Writer, objs []runtime.Object) error {
	data, err := yaml.Export(objs...)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}

// Objects returns runtime.Objects for every CRD or CRD Dependency this operator relies on
func Objects(v1beta1 bool) (crds, crdDeps []runtime.Object, err error) {
	crdDefs, crdDepDefs := List()
	crds, err = objects(v1beta1, crdDefs)
	if err != nil {
		return nil, nil, err
	}
	crdDeps, err = objects(v1beta1, crdDepDefs)
	if err != nil {
		return nil, nil, err
	}
	return
}

func objects(v1beta1 bool, crdDefs []crd.CRD) (crds []runtime.Object, err error) {
	for _, crdDef := range crdDefs {
		if v1beta1 {
			crd, err := crdDef.ToCustomResourceDefinitionV1Beta1()
			if err != nil {
				return nil, err
			}
			crds = append(crds, crd)
		} else {
			crd, err := crdDef.ToCustomResourceDefinition()
			if err != nil {
				return nil, err
			}
			crds = append(crds, crd)
		}
	}
	return
}

// List returns the list of CRDs and dependent CRDs for this operator
func List() ([]crd.CRD, []crd.CRD) {
	crds := []crd.CRD{
		newCRD(
			"ProjectHelmChart.helm.cattle.io/v1alpha1",
			&v1alpha1.ProjectHelmChart{},
			func(c crd.CRD) crd.CRD {
				return c.
					WithColumn("Status", ".status.status").
					WithColumn("System Namespace", ".status.systemNamespace").
					WithColumn("Release Namespace", ".status.releaseNamespace").
					WithColumn("Release Name", ".status.releaseName").
					WithColumn("Target Namespaces", ".status.targetNamespaces")
			},
		),
	}
	crdDeps := append(helmcontrollercrd.List(), helmlockercrd.List()...)
	return crds, crdDeps
}

// Create creates all CRDs and dependent CRDs in the cluster
func Create(ctx context.Context, cfg *rest.Config) error {
	factory, err := crd.NewFactoryFromClient(cfg)
	if err != nil {
		return err
	}

	crdClientSet := factory.CRDClient.(*clientset.Clientset)
	crds, crdDeps := List()
	// TODO: CRD updates topic, We could opt to not filter `crds` value and always install those?
	err = filterMissingCRDs(crdClientSet, &crds)
	if err != nil {
		return err
	}

	err = filterMissingCRDs(crdClientSet, &crdDeps)
	if err != nil {
		return err
	}

	return factory.BatchCreateCRDs(ctx, append(crds, crdDeps...)...).BatchWait()
}

func newCRD(namespacedType string, obj interface{}, customize func(crd.CRD) crd.CRD) crd.CRD {
	newCrd := crd.NamespacedType(namespacedType).
		WithSchemaFromStruct(obj).
		WithStatus()

	if customize != nil {
		newCrd = customize(newCrd)
	}
	return newCrd
}

// filterMissingCRDs takes a list of expected CRDs and returns a filtered list of missing CRDs.
func filterMissingCRDs(apiExtClient *clientset.Clientset, expectedCRDs *[]crd.CRD) error {
	for i := len(*expectedCRDs) - 1; i >= 0; i-- {
		currentCRD := (*expectedCRDs)[i]
		crdName := currentCRD.GVK.GroupVersion().WithKind(strings.ToLower(name.GuessPluralName(currentCRD.GVK.Kind))).GroupKind().String()

		// try to get the given CRD just to check for error, verifying if it exists
		foundCRD, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), crdName, metav1.GetOptions{})
		logrus.Debugf(
			"Found `%s` at version `%s`, expecting version `%s`",
			crdName,
			foundCRD.Status.StoredVersions[0],
			currentCRD.GVK.Version,
		)
		if err == nil {
			logrus.Debugf("Installing `%s` will be skipped; a suitible version exists on the cluster", crdName)
			// Update the list to remove the current item since the CRD is in the cluster already
			*expectedCRDs = append((*expectedCRDs)[:i], (*expectedCRDs)[i+1:]...)
		} else if !errors.IsNotFound(err) {
			*expectedCRDs = []crd.CRD{}
			return fmt.Errorf("failed to check CRD %s: %v", crdName, err)
		} else {
			logrus.Debugf("`%s` will be installed or updated", crdName)
		}
	}

	return nil
}
