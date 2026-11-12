package common

import (
	"errors"

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

	if len(opts.ManagedChartAuthSecret) > 0 && (len(opts.ManagedChartName) == 0 || len(opts.ManagedChartRepo) == 0 || len(opts.ManagedChartVersion) == 0) {
		return errors.New("must provide --managed-chart-name, --managed-chart-repo, and --managed-chart-version when --managed-chart-auth-secret is set")
	}

	if opts.UsesManagedChartReference() {
		switch {
		case len(opts.ManagedChartName) == 0:
			return errors.New("must provide --managed-chart-name when configuring an approved managed chart reference")
		case len(opts.ManagedChartRepo) == 0:
			return errors.New("must provide --managed-chart-repo when configuring an approved managed chart reference")
		case len(opts.ManagedChartVersion) == 0:
			return errors.New("must provide --managed-chart-version when configuring an approved managed chart reference")
		}
	} else if len(opts.ChartContent) == 0 {
		return errors.New("cannot instantiate Project Operator without either embedded chart content or an approved managed chart reference")
	}

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

func (opts Options) UsesManagedChartReference() bool {
	return len(opts.ManagedChartName) > 0 || len(opts.ManagedChartRepo) > 0 || len(opts.ManagedChartVersion) > 0
}
