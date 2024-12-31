package main

import (
	"os"

	"github.com/rancher/helmcommon/pkg/test/apis"
	"github.com/rancher/wrangler/v3/pkg/cleanup"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
	"github.com/sirupsen/logrus"
)

func main() {
	cleanupTarget()
	generate()
}

func generate() {
	logrus.Info("Generating controller boilerplate")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/helmcommon/pkg/test/generated",
		Boilerplate:   "../../gen/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"test.cattle.io": {
				Types: []interface{}{
					apis.A{},
				},
				GenerateTypes: true,
			},
		},
	})
}

func cleanupTarget() {
	if err := cleanup.Cleanup("./pkg/test/apis"); err != nil {
		logrus.Fatal(err)
	}
	if err := os.RemoveAll("./pkg/test/generated"); err != nil {
		logrus.Fatal(err)
	}
}
