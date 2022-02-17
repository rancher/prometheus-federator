package main

import (
	"fmt"
	"os"

	// v1alpha1 "github.com/aiyengar2/prometheus-federator/pkg/apis/monitoring.cattle.io/v1alpha1"
	"github.com/aiyengar2/prometheus-federator/pkg/apis/monitoring.cattle.io/v1alpha1"
	"github.com/aiyengar2/prometheus-federator/pkg/crd"
	prometheusoperator "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "crds" {
		fmt.Println("Writing CRDs to", os.Args[2])
		if err := crd.WriteFile(os.Args[2]); err != nil {
			panic(err)
		}
		return
	}

	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/aiyengar2/prometheus-federator/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"monitoring.cattle.io": {
				Types: []interface{}{
					v1alpha1.Project{},
				},
				GenerateTypes: true,
			},
			"monitoring.coreos.com": {
				Types: []interface{}{
					prometheusoperator.Prometheus{},
					prometheusoperator.Alertmanager{},
					prometheusoperator.PrometheusRule{},
					prometheusoperator.PodMonitor{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
		},
	})
}
