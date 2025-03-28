package main

import (
	"os"

	helmcontrollercrd "github.com/k3s-io/helm-controller/pkg/crd"
	lockercrd "github.com/rancher/prometheus-federator/internal/helm-locker/crd"
	helmprojectcrds "github.com/rancher/prometheus-federator/internal/helm-project-operator/crd"
	commoncrds "github.com/rancher/prometheus-federator/internal/helmcommon/pkg/crds"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.Info("Cleaning up CRDs")
	if err := os.RemoveAll("./crds"); err != nil {
		logrus.Fatal(err)
	}

	crdPath := "./crds"
	depCrdPath := "./crds/dep"
	tracker := commoncrds.NewCRDTracker(
		append(
			lockercrd.Required(),
			helmprojectcrds.Required()...,
		),
	)

	depTracker := commoncrds.NewCRDTracker(
		helmcontrollercrd.List(),
	)
	logrus.Infof("Writing CRDS to %s", crdPath)
	if err := commoncrds.WriteFiles(tracker, crdPath); err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("Writing CRD dependencies to %s", depCrdPath)
	if err := commoncrds.WriteFiles(depTracker, depCrdPath); err != nil {
		logrus.Fatal(err)
	}

}
