package main

import (
	"os"

	"github.com/rancher/wrangler/pkg/cleanup"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := cleanup.Cleanup("./internal/helm-locker/apis"); err != nil {
		logrus.Fatal(err)
	}
	if err := os.RemoveAll("./internal/helm-locker/generated"); err != nil {
		logrus.Fatal(err)
	}
	if err := os.RemoveAll("./crds/helm-locker"); err != nil {
		logrus.Fatal(err)
	}
}
