package main

import (
	"os"

	"github.com/rancher/wrangler/pkg/cleanup"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := cleanup.Cleanup("./pkg/helm-project-operator/apis"); err != nil {
		logrus.Fatal(err)
	}
	if err := os.RemoveAll("./pkg/helm-project-operator/generated"); err != nil {
		logrus.Fatal(err)
	}
	if err := os.RemoveAll("./crds/helm-project-operator"); err != nil {
		logrus.Fatal(err)
	}
}
