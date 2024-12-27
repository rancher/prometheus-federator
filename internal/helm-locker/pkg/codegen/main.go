package main

import (
	"os"

	v1alpha1 "github.com/rancher/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/helm-locker/pkg/crd"
	"github.com/sirupsen/logrus"

	"github.com/rancher/wrangler/v3/pkg/cleanup"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
)

func main() {
	logrus.Infof("Generating code")
	if err := cleanup.Cleanup("./pkg/apis"); err != nil {
		logrus.Fatal(err)
	}

	if err := os.RemoveAll("./pkg/generated"); err != nil {
		logrus.Fatal(err)
	}

	if err := os.RemoveAll("./crds"); err != nil {
		logrus.Fatal(err)
	}

	os.Unsetenv("GOPATH")

	logrus.Info("Generating controller boilerplate")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/helm-locker/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"helm.cattle.io": {
				Types: []interface{}{
					v1alpha1.HelmRelease{},
				},
				GenerateTypes: true,
			},
		},
	})
	crdPath := "./crds/crds.yaml"
	logrus.Infof("Writing CRDS to %s", crdPath)
	if err := crd.WriteFile(crdPath); err != nil {
		logrus.Fatal(err)
	}
}
