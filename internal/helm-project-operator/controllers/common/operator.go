package common

import (
	"errors"

	"github.com/sirupsen/logrus"
)

// OperatorOptions are options provided by an operator that is implementing Helm Project Operator
type OperatorOptions struct {
	// HelmAPIVersion is the unique API version marking ProjectHelmCharts that this Helm Project Operator should watch for
	HelmAPIVersion string

	// ReleaseName is a name that identifies releases created for this operator
	ReleaseName string

	// SystemNamespaces are additional operator namespaces to treat as if they are system namespaces whether or not
	// they are marked via some sort of annotation
	SystemNamespaces []string

	// ChartContent is the base64 tgz contents of the folder containing the Helm chart that needs to be deployed
	ChartContent string

	// Singleton marks whether only a single ProjectHelmChart can exist per registration namespace
	// If enabled, it will ensure that releases are named based on the registration namespace rather than
	// the name provided on the ProjectHelmChart, which is what triggers an UnableToCreateHelmRelease status
	// on the ProjectHelmChart created after this one
	Singleton bool
}

// Validate validates the provided OperatorOptions
func (opts OperatorOptions) Validate() error {
	if len(opts.HelmAPIVersion) == 0 {
		return errors.New("must provide a spec.helmApiVersion that this project operator is being initialized for")
	}

	if len(opts.ReleaseName) == 0 {
		return errors.New("must provide name of Helm release that this project operator should deploy")
	}

	if len(opts.SystemNamespaces) > 0 {
		logrus.Infof("Marking the following namespaces as system namespaces: %s", opts.SystemNamespaces)
	}

	if len(opts.ChartContent) == 0 {
		return errors.New("cannot instantiate Project Operator without bundling a Helm chart to provide for the HelmChart's spec.ChartContent")
	}

	return nil
}
