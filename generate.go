//go:generate go run pkg/codegen/buildconfig/writer.go pkg/codegen/buildconfig/main.go

//go:generate go run go.uber.org/mock/mockgen -destination=internal/test/mocks/mock_apiextension.go -package mock "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1" CustomResourceDefinitionInterface
//go:generate go run go.uber.org/mock/mockgen -destination=internal/test/mocks/mock_node.go -package mock "k8s.io/client-go/kubernetes/typed/core/v1" NodeInterface

//go:generate go run internal/codegen/crd/main.go
//go:generate go run internal/codegen/main.go
package main
