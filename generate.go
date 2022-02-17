//go:generate go run pkg/codegen/cleanup/main.go
//go:generate go run pkg/codegen/main.go
//go:generate go run ./pkg/codegen crds ./charts/prometheus-federator-crd/templates/crds.yaml

package main
