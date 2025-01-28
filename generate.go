//go:generate go run pkg/codegen/buildconfig/writer.go pkg/codegen/buildconfig/main.go

//go:generate go run internal/helm-locker/codegen/cleanup/main.go
//go:generate go run internal/helm-locker/codegen/main.go
//go:generate go run ./internal/helm-locker/codegen crds ./crds/helm-locker/crds.yaml

//go:generate go run internal/helm-project-operator/codegen/cleanup/main.go
//go:generate go run internal/helm-project-operator/codegen/main.go
//go:generate go run ./internal/helm-project-operator/codegen crds ./crds/helm-project-operator ./crds/helm-project-operator

package main
