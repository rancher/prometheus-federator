package main

import (
	"os"

	v1alpha1 "github.com/rancher/helm-project-operator/pkg/helm-locker/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/helm-project-operator/pkg/helm-locker/crd"
	"github.com/sirupsen/logrus"

	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "crds" {
		if len(os.Args) != 3 {
			logrus.Fatal("usage: ./codegen crds <crd-directory>")
		}
		logrus.Infof("Writing CRDs to %s", os.Args[2])
		if err := crd.WriteFile(os.Args[2]); err != nil {
			panic(err)
		}
		return
	}

	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/helm-project-operator/pkg/helm-locker/generated",
		Boilerplate:   "gen/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"helm.cattle.io": {
				Types: []interface{}{
					v1alpha1.HelmRelease{},
				},
				GenerateTypes: true,
			},
		},
	})
}
