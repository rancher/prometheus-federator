package common

const (
	// ProjectRegistrationNamespaceFmt is the format used in order to create project registration namespaces if ProjectLabel is provided
	// If SystemProjectLabel is also provided, the project release namespace will be this namespace with `-<ReleaseName>` suffixed, where
	// ReleaseName is provided by the Project Operator that implements Helm Project Operator
	ProjectRegistrationNamespaceFmt = "cattle-project-%s"
)
