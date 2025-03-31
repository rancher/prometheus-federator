package common

import (
	"os"
	"path/filepath"

	"github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type RuntimeOptions struct {
	// Namespace is the systemNamespace to create HelmCharts and HelmReleases in
	// It's generally expected that this namespace is not widely accessible by all users in your cluster; it's recommended that it is placed
	// in something akin to a System Project that is locked down in terms of permissions since resources like HelmCharts and HelmReleases are deployed there
	Namespace string `usage:"Namespace to create HelmCharts and HelmReleases; if ProjectLabel is not provided, this will also be the namespace to watch ProjectHelmCharts" default:"cattle-helm-system" env:"NAMESPACE"`

	// NodeName is the name of the node running the operator; it adds additional information to events about where they were generated from
	NodeName string `usage:"Name of the node this controller is running on" env:"NODE_NAME"`

	// ControllerName is the name of the controller that identifies this operator; this ensures that all HelmCharts and HelmReleases have the correct managed-by annotation
	// so that multiple iterations of this operator in the same namespace do not try to manage the same HelmChart and HelmRelease objects
	ControllerName string `usage:"Unique name to identify this controller that is added to all HelmCharts tracked by this controller" default:"helm-project-operator" env:"CONTROLLER_NAME"`

	// HelmJobImage is the job image to use to run the HelmChart job (default rancher/klipper-helm:v0.9.4-build20250113)
	// Generally, this HelmJobImage can be left undefined, but may be necessary to be set if you are running with a non-default image
	HelmJobImage string `usage:"Job image to use to perform helm operations on HelmChart creation" env:"HELM_JOB_IMAGE"`

	// ClusterID identifies the cluster that the operator is being operated frmo within; it adds an additional annotation to project registration
	// namespaces that indicates the projectID with the cluster label.
	//
	// Note: primarily used for integration with Rancher Projects
	ClusterID string `usage:"Identifies the cluster this controller is running on. Ignored if --project-label is not provided." env:"CLUSTER_ID"`

	// SystemDefaultRegistry is the prefix to be added to all images deployed by the HelmChart embedded into the Project Operator
	// to point at the right set of images that need to be deployed. This is usually provided in Rancher as global.cattle.systemDefaultRegistry
	SystemDefaultRegistry string `usage:"Default system registry to use for Docker images deployed by underlying Helm Chart. Provided as global.cattle.systemDefaultRegistry in the Helm Chart" env:"SYSTEM_DEFAULT_REGISTRY"`

	// CattleURL is the Rancher URL that this chart has been deployed onto. This is usually provided in Rancher Helm charts as global.cattle.url
	CattleURL string `usage:"Default Rancher URL to provide to the Helm chart under global.cattle.url" env:"CATTLE_URL"`

	// ProjectLabel is the label that identifies projects
	// Note: this field is optional and ensures that ProjectHelmCharts auto-infer their spec.projectNamespaceSelector
	// If provided, any spec.projectNamespaceSelector provided will be ignored
	// example: field.cattle.io/projectId
	ProjectLabel string `usage:"Label on namespaces to create Project Registration Namespaces and watch for ProjectHelmCharts" env:"PROJECT_LABEL"`

	// SystemProjectLabelValues are values of ProjectLabel that identify system namespaces. Does nothing if ProjectLabel is not provided
	// example: p-ranch
	// If both this and the ProjectLabel example are provided, any namespaces with label 'field.cattle.io/projectId: <system-project-label-value>'
	// will be treated as a systemNamespace, which means that no ProjectHelmChart will be allowed to select it
	SystemProjectLabelValues []string `usage:"Values on project label on namespaces that marks it as a system namespace" env:"SYSTEM_PROJECT_LABEL_VALUE"`

	// ProjectReleaseLabelValue is the value of the ProjectLabel that should be added to Project Release Namespaces. Does nothing if ProjectLabel is not provided
	// example: p-ranch
	// If provided, dedicated Project Release namespaces will be created in the cluster for each ProjectHelmChart that needs a Helm Release
	// The created Project Release namespaces will also automatically be identified as a System Project Namespaces based on this label, so other
	// namespaces with this label value will be treated as a system namespace as well
	ProjectReleaseLabelValue string `usage:"Value on project label on namespaces that marks it as a system namespace" env:"SYSTEM_PROJECT_LABEL_VALUE"`

	// AdminClusterRole configures the operator to automaticaly create RoleBindings on Roles in the Project Release Namespace marked with
	// 'helm.cattle.io/project-helm-chart-role': '<helm-release>' and 'helm.cattle.io/project-helm-chart-role-aggregate-from': 'admin'
	// based on ClusterRoleBindings or RoleBindings in the Project Registration namespace tied to the provided ClusterRole, if it exists
	AdminClusterRole string `usage:"ClusterRole tied to admin users who should have permissions in the Project Release Namespace" env:"ADMIN_CLUSTER_ROLE"`

	// EditClusterRole configures the operator to automaticaly create RoleBindings on Roles in the Project Release Namespace marked with
	// 'helm.cattle.io/project-helm-chart-role': '<helm-release>' and 'helm.cattle.io/project-helm-chart-role-aggregate-from': 'edit'
	// based on ClusterRoleBindings or RoleBindings in the Project Registration namespace tied to the provided ClusterRole, if it exists
	EditClusterRole string `usage:"ClusterRole tied to edit users who should have permissions in the Project Release Namespace" env:"EDIT_CLUSTER_ROLE"`

	// ViewClusterRole configures the operator to automaticaly create RoleBindings on Roles in the Project Release Namespace marked with
	// 'helm.cattle.io/project-helm-chart-role': '<helm-release>' and 'helm.cattle.io/project-helm-chart-role-aggregate-from': 'view'
	// based on ClusterRoleBindings or RoleBindings in the Project Registration namespace tied to the provided ClusterRole, if it exists
	ViewClusterRole string `usage:"ClusterRole tied to view users who should have permissions in the Project Release Namespace" env:"VIEW_CLUSTER_ROLE"`

	// DisableHardening turns off the controller that manages the default service account and a default NetworkPolicy deployed on all
	// namespaces marked with the Helm Project Operated Label to prevent generated namespaces from breaking a CIS 1.16 Hardened Scan by patching
	// the default ServiceAccount and creating a default secure NetworkPolicy.
	//
	// ref: https://docs.rke2.io/security/cis_self_assessment16/#515
	// ref: https://docs.rke2.io/security/cis_self_assessment16/#532
	//
	// To configure the default ServiceAccount and NetworkPolicy across all generated namespaces, you can provide overrides in the HardeningOptionsFile
	// If you need to configure the default ServiceAccount and NetworkPolicy on a per-namespace basis, it is recommended that you disable this
	DisableHardening bool `usage:"Path to file that contains the configuration for the default ServiceAccount and NetworkPolicy deployed on operated namespaces" env:"HARDENING_OPTIONS_FILE"`

	// HardeningOptionsFile is the path to the file that contains the configuration for the default ServiceAccount and NetworkPolicy deployed on operated namespaces
	// By default, the default service account of the namespace is patched to disable automountServiceAccountToken
	// By default, a default NetworkPolicy is deployed in the namespace that selects all pods in the namespace and limits all ingress and egress
	HardeningOptionsFile string `usage:"Path to file that contains the configuration for the default ServiceAccount and NetworkPolicy deployed on operated namespaces" default:"hardening.yaml" env:"HARDENING_OPTIONS_FILE"`

	// ValuesOverrideFile is the path to the file that contains operated-provided overrides on the values.yaml that should be applied for each ProjectHelmChart
	ValuesOverrideFile string `usage:"Path to file that contains values.yaml overrides supplied by the operator" default:"values.yaml" env:"VALUES_OVERRIDE_FILE"`

	// DisableEmbeddedHelmLocker determines whether to disable embedded Helm Locker controller in favor of external Helm Locker
	DisableEmbeddedHelmLocker bool `usage:"Whether to disable embedded Helm Locker controller in favor of external Helm Locker" env:"DISABLE_EMBEDDED_HELM_LOCKER"`

	// DisableEmbeddedHelmController determines whether to disable embedded Helm Controller controller in favor of external Helm Controller
	// This should be the default in most RKE2 clusters since the RKE2 server binary already embeds a Helm Controller instance that manages HelmCharts
	DisableEmbeddedHelmController bool `usage:"Whether to disable embedded Helm Controller controller in favor of external Helm Controller (recommended for RKE2 clusters)" env:"DISABLE_EMBEDDED_HELM_CONTROLLER"`
}

// Validate validates the provided RuntimeOptions
func (opts RuntimeOptions) Validate() error {
	if len(opts.ProjectLabel) > 0 {
		logrus.Infof("Creating dedicated project registration namespaces to discover ProjectHelmCharts based on the value found for the project label '%s' on all namespaces in the cluster, excluding system namespaces; these namespaces will need to be manually cleaned up if they have the label '%s': 'true'", opts.ProjectLabel, HelmProjectOperatedNamespaceOrphanedLabel)
		if len(opts.SystemProjectLabelValues) > 0 {
			for _, systemProjectLabel := range opts.SystemProjectLabelValues {
				logrus.Infof("Assuming namespaces tagged with %s=%s are also system namespaces", opts.ProjectLabel, systemProjectLabel)
			}
		}
		if len(opts.ProjectReleaseLabelValue) > 0 {
			logrus.Infof("Assuming namespaces tagged with %s=%s are also system namespaces", opts.ProjectLabel, opts.ProjectReleaseLabelValue)
			logrus.Infof("Creating dedicated project release namespaces for ProjectHelmCharts with label '%s': '%s'; these namespaces will need to be manually cleaned up if they have the label '%s': 'true'", opts.ProjectLabel, opts.ProjectReleaseLabelValue, HelmProjectOperatedNamespaceOrphanedLabel)
		}
		if len(opts.ClusterID) > 0 {
			logrus.Infof("Marking project registration namespaces with %s=%s:<projectID>", opts.ProjectLabel, opts.ClusterID)
		}
	}

	if len(opts.HelmJobImage) > 0 {
		logrus.Infof("Using %s as spec.JobImage on all generated HelmChart resources", opts.HelmJobImage)
	}

	if len(opts.NodeName) > 0 {
		logrus.Infof("Marking events as being sourced from node %s", opts.NodeName)
	}

	if opts.DisableHardening {
		logrus.Info("Hardening is disabled")
	} else {
		logrus.Info("Managing the configuration of the default ServiceAccount and an auto-generated NetworkPolicy in all namespaces managed by this Project Operator")
	}

	return nil
}

// LoadValuesOverrideFromFile unmarshalls the struct found at the file to YAML and reads it into memory
func LoadValuesOverrideFromFile(path string) (v1alpha1.GenericMap, error) {
	var valuesOverride v1alpha1.GenericMap
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	abspath := filepath.Join(wd, path)
	_, err = os.Stat(abspath)
	if err != nil {
		if os.IsNotExist(err) {
			// we just assume the default is used
			err = nil
		}
		return nil, err
	}
	valuesOverrideBytes, err := os.ReadFile(abspath)
	if err != nil {
		return nil, err
	}
	return valuesOverride, yaml.Unmarshal(valuesOverrideBytes, &valuesOverride)
}
