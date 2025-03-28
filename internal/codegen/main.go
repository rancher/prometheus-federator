package main

import (
	"os"

	locker_v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-locker/apis/helm.cattle.io/v1alpha1"
	hpo_v1alpha1 "github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/cleanup"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.Info("Generating code for helm-locker")

	if err := cleanup.Cleanup("./internal/helm-locker/apis"); err != nil {
		logrus.Fatal(err)
	}

	if err := os.RemoveAll("./internal/helm-locker/generated"); err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Generating controller boilerplate")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/prometheus-federator/internal/helm-locker/generated",
		Boilerplate:   "./gen/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"helm.cattle.io": {
				Types: []interface{}{
					locker_v1alpha1.HelmRelease{},
				},
				GenerateTypes: true,
			},
		},
	})

	logrus.Info("Genering code for helm-project-operator")
	if err := cleanup.Cleanup("./internal/helm-project-operator/apis"); err != nil {
		logrus.Fatal(err)
	}

	if err := os.RemoveAll("./internal/helm-project-operator/generated"); err != nil {
		logrus.Fatal(err)
	}

	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/prometheus-federator/internal/helm-project-operator/generated",
		Boilerplate:   "./gen/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"helm.cattle.io": {
				Types: []interface{}{
					hpo_v1alpha1.ProjectHelmChart{},
				},
				GenerateTypes: true,
			},
		},
	})
}
