package common

import (
	"github.com/sirupsen/logrus"
)

// Options defines options that can be set on initializing the HelmProjectOperator
type Options struct {
	RuntimeOptions
	OperatorOptions
}

// Validate validates the provided Options
func (opts Options) Validate() error {
	if err := opts.OperatorOptions.Validate(); err != nil {
		return err
	}

	if err := opts.RuntimeOptions.Validate(); err != nil {
		return err
	}

	// Cross option checks

	if opts.Singleton {
		logrus.Infof("Note: Operator only supports a single ProjectHelmChart per project registration namespace")
		if len(opts.ProjectLabel) == 0 {
			logrus.Warnf("It is only recommended to run a singleton Project Operator when --project-label is provided (currently not set). The current configuration of this operator would only allow a single ProjectHelmChart to be managed by this Operator.")
		}
	}

	for subjectRole, defaultClusterRoleName := range GetDefaultClusterRoles(opts) {
		logrus.Infof("RoleBindings will automatically be created for Roles in the Project Release Namespace marked with '%s': '<helm-release>' "+
			"and '%s': '%s' based on ClusterRoleBindings or RoleBindings in the Project Registration namespace tied to ClusterRole %s",
			HelmProjectOperatorProjectHelmChartRoleLabel, HelmProjectOperatorProjectHelmChartRoleAggregateFromLabel, subjectRole, defaultClusterRoleName,
		)
	}

	return nil
}
