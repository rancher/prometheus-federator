package main

import (
	"os"

	v1alpha1 "github.com/rancher/helm-locker/pkg/apis/helm.cattle.io/v1alpha1"
	lockercrd "github.com/rancher/helm-locker/pkg/crd"
	commoncrds "github.com/rancher/helmcommon/pkg/crds"
	"github.com/rancher/wrangler/v3/pkg/cleanup"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
	"github.com/sirupsen/logrus"
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
	tracker := commoncrds.NewCRDTracker(
		lockercrd.Required(),
	)

	if err := commoncrds.WriteFiles(tracker, crdPath); err != nil {
		logrus.Fatal(err)
	}
}
