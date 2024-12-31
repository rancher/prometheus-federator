//go:generate go run internal/helm-locker/pkg/codegen/main.go
//go:generate go run internal/helm-locker/pkg/codegen/main.go ./crds/helm-locker/crds.yaml
//go:generate go run internal/helm-project-operator/pkg/codegen/main.go
//go:generate go run internal/helm-project-operator/pkg/codegen/main.go ./crds/helm-project-operator ./crds/helm-project-operator

package main
