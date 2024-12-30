package main

import (
	"os"

	v1alpha1 "github.com/rancher/helm-project-operator/pkg/apis/helm.cattle.io/v1alpha1"
	helmprojectcrds "github.com/rancher/helm-project-operator/pkg/crd"
	commoncrds "github.com/rancher/helmcommon/pkg/crds"
	"github.com/sirupsen/logrus"

	"github.com/rancher/wrangler/v3/pkg/cleanup"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
)

func main() {
	cleanupGen()
	generate()
}

func generate() {
	if len(os.Args) > 3 && os.Args[1] == "crds" {
		required := commoncrds.NewCRDTracker(
			helmprojectcrds.Required(),
		)

		dependencies := commoncrds.NewCRDTracker(
			append(
				helmprojectcrds.RequiredEmbedded(),
				helmprojectcrds.RequiredDependencies()...,
			),
		)
		if len(os.Args) != 4 {
			logrus.Fatal("usage: ./codegen crds <crd-directory> <crd-dependency-directory>")
		}
		crdPath, crdDepPath := os.Args[2], os.Args[3]
		logrus.Infof("Writing required CRDs to %s", crdPath)
		if err := commoncrds.WriteFiles(required, crdPath); err != nil {
			panic(err)
		}
		logrus.Infof("Writing dependent CRDs to %s", crdDepPath)
		if err := commoncrds.WriteFiles(dependencies, crdDepPath); err != nil {
			panic(err)
		}
		return
	}

	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/helm-project-operator/pkg/generated",
		Boilerplate:   "../gen/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"helm.cattle.io": {
				Types: []interface{}{
					v1alpha1.ProjectHelmChart{},
				},
				GenerateTypes: true,
			},
		},
	})
}

func cleanupGen() {
	if err := cleanup.Cleanup("./pkg/apis"); err != nil {
		logrus.Fatal(err)
	}
	if err := os.RemoveAll("./pkg/generated"); err != nil {
		logrus.Fatal(err)
	}
	if err := os.RemoveAll("./crds"); err != nil {
		logrus.Fatal(err)
	}
}
