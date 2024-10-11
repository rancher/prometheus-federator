//go:generate go run pkg/helm-locker/codegen/cleanup/main.go
//go:generate go run pkg/helm-locker/codegen/main.go
//go:generate go run ./pkg/helm-locker/codegen crds ./crds/helm-locker/crds.yaml

//go:generate go run pkg/helm-project-operator/codegen/cleanup/main.go
//go:generate go run pkg/helm-project-operator/codegen/main.go
//go:generate go run ./pkg/helm-project-operator/codegen crds ./crds/helm-project-operator ./crds/helm-project-operator

package main
