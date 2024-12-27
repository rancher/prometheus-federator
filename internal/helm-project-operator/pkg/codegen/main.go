package main

import (
	"os"

	v1alpha1 "github.com/rancher/helm-project-operator/pkg/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/helm-project-operator/pkg/crd"
	"github.com/sirupsen/logrus"

	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
)

func main() {
	if len(os.Args) > 3 && os.Args[1] == "crds" {
		if len(os.Args) != 4 {
			logrus.Fatal("usage: ./codegen crds <crd-directory> <crd-dependency-directory>")
		}
		logrus.Infof("Writing CRDs to %s and %s", os.Args[2], os.Args[3])
		if err := crd.WriteFiles(os.Args[2], os.Args[3]); err != nil {
			panic(err)
		}
		return
	}

	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/helm-project-operator/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
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
