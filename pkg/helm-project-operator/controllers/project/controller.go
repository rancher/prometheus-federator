package project

import (
	"context"
	"fmt"

	v1alpha2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	common2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"
	"github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/namespace"
	helmprojectcontroller "github.com/rancher/prometheus-federator/pkg/helm-project-operator/generated/controllers/helm.cattle.io/v1alpha1"

	"github.com/k3s-io/helm-controller/pkg/controllers/chart"
	k3shelmcontroller "github.com/k3s-io/helm-controller/pkg/generated/controllers/helm.cattle.io/v1"
	helmlockercontroller "github.com/rancher/prometheus-federator/pkg/helm-locker/generated/controllers/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/pkg/remove"
	"github.com/rancher/wrangler/pkg/apply"
	corecontroller "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	rbaccontroller "github.com/rancher/wrangler/pkg/generated/controllers/rbac/v1"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	DefaultJobImage = chart.DefaultJobImage
)

type handler struct {
	systemNamespace         string
	opts                    common2.Options
	valuesOverride          v1alpha2.GenericMap
	apply                   apply.Apply
	projectHelmCharts       helmprojectcontroller.ProjectHelmChartController
	projectHelmChartCache   helmprojectcontroller.ProjectHelmChartCache
	configmaps              corecontroller.ConfigMapController
	configmapCache          corecontroller.ConfigMapCache
	roles                   rbaccontroller.RoleController
	roleCache               rbaccontroller.RoleCache
	clusterrolebindings     rbaccontroller.ClusterRoleBindingController
	clusterrolebindingCache rbaccontroller.ClusterRoleBindingCache
	helmCharts              k3shelmcontroller.HelmChartController
	helmReleases            helmlockercontroller.HelmReleaseController
	namespaces              corecontroller.NamespaceController
	namespaceCache          corecontroller.NamespaceCache
	rolebindings            rbaccontroller.RoleBindingController
	rolebindingCache        rbaccontroller.RoleBindingCache
	projectGetter           namespace.ProjectGetter
}

func Register(
	ctx context.Context,
	systemNamespace string,
	opts common2.Options,
	valuesOverride v1alpha2.GenericMap,
	apply apply.Apply,
	projectHelmCharts helmprojectcontroller.ProjectHelmChartController,
	projectHelmChartCache helmprojectcontroller.ProjectHelmChartCache,
	configmaps corecontroller.ConfigMapController,
	configmapCache corecontroller.ConfigMapCache,
	roles rbaccontroller.RoleController,
	roleCache rbaccontroller.RoleCache,
	clusterrolebindings rbaccontroller.ClusterRoleBindingController,
	clusterrolebindingCache rbaccontroller.ClusterRoleBindingCache,
	helmCharts k3shelmcontroller.HelmChartController,
	helmReleases helmlockercontroller.HelmReleaseController,
	namespaces corecontroller.NamespaceController,
	namespaceCache corecontroller.NamespaceCache,
	rolebindings rbaccontroller.RoleBindingController,
	rolebindingCache rbaccontroller.RoleBindingCache,
	projectGetter namespace.ProjectGetter,
) {

	apply = apply.
		// Why do we need the release name?
		// To ensure that we don't override the set created by another instance of the Project Operator
		// running under a different release name operating on the same project registration namespace
		WithSetID(fmt.Sprintf("%s-project-helm-chart-applier", opts.ReleaseName)).
		WithCacheTypes(
			helmCharts,
			helmReleases,
			namespaces,
			rolebindings).
		WithNoDeleteGVK(namespaces.GroupVersionKind())

	h := &handler{
		systemNamespace:         systemNamespace,
		opts:                    opts,
		valuesOverride:          valuesOverride,
		apply:                   apply,
		projectHelmCharts:       projectHelmCharts,
		projectHelmChartCache:   projectHelmChartCache,
		configmaps:              configmaps,
		configmapCache:          configmapCache,
		roles:                   roles,
		clusterrolebindings:     clusterrolebindings,
		clusterrolebindingCache: clusterrolebindingCache,
		roleCache:               roleCache,
		helmCharts:              helmCharts,
		helmReleases:            helmReleases,
		namespaces:              namespaces,
		namespaceCache:          namespaceCache,
		rolebindings:            rolebindings,
		rolebindingCache:        rolebindingCache,
		projectGetter:           projectGetter,
	}

	h.initIndexers()

	h.initResolvers(ctx)

	// Why do we need to add the managedBy string to the generatingHandlerName?
	//
	// By default, generating handlers use the name of the controller as the set ID for the wrangler.apply operation
	// Therefore, if multiple iterations of the helm-controller are using the same set ID, they will try to overwrite each other's
	// resources since each controller will detect the other's set as resources that need to be cleaned up to apply the new set
	//
	// To resolve this, we simply prefix the provided managedBy string to the generatingHandler controller's name only to ensure that the
	// set ID specified will only target this particular controller
	generatingHandlerName := fmt.Sprintf("%s-project-helm-chart-registration", opts.ControllerName)
	helmprojectcontroller.RegisterProjectHelmChartGeneratingHandler(ctx,
		projectHelmCharts,
		apply,
		"",
		generatingHandlerName,
		h.OnChange,
		&generic.GeneratingHandlerOptions{
			AllowClusterScoped: true,
		})

	remove.RegisterScopedOnRemoveHandler(ctx, projectHelmCharts, "on-project-helm-chart-remove",
		func(_ string, obj runtime.Object) (bool, error) {
			if obj == nil {
				return false, nil
			}
			projectHelmChart, ok := obj.(*v1alpha2.ProjectHelmChart)
			if !ok {
				return false, nil
			}
			return h.shouldManage(projectHelmChart), nil
		},
		helmprojectcontroller.FromProjectHelmChartHandlerToHandler(h.OnRemove),
	)

	err := h.initRemoveCleanupLabels()
	if err != nil {
		logrus.Fatal(err)
	}
}

func (h *handler) shouldManage(projectHelmChart *v1alpha2.ProjectHelmChart) bool {
	if projectHelmChart == nil {
		return false
	}
	namespace, err := h.namespaceCache.Get(projectHelmChart.Namespace)
	if err != nil {
		// If the namespace that the projectHelmChart resides in does not exist, it shouldn't be managed
		//
		// Note: we know that this error would only happen if the namespace is not found since the only valid error returned from this
		// call is errors.NewNotFound(c.resource, name)
		return false
	}
	isProjectRegistrationNamespace := h.projectGetter.IsProjectRegistrationNamespace(namespace)
	if !isProjectRegistrationNamespace {
		// only watching resources in registered namespaces
		return false
	}
	if projectHelmChart.Spec.HelmAPIVersion != h.opts.HelmAPIVersion {
		// only watch resources with the HelmAPIVersion this controller was configured with
		return false
	}
	return true
}

func (h *handler) OnChange(projectHelmChart *v1alpha2.ProjectHelmChart, projectHelmChartStatus v1alpha2.ProjectHelmChartStatus) ([]runtime.Object, v1alpha2.ProjectHelmChartStatus, error) {
	var objs []runtime.Object

	// initial checks to see if we should handle this
	shouldManage := h.shouldManage(projectHelmChart)
	if !shouldManage {
		return nil, projectHelmChartStatus, nil
	}
	if projectHelmChart.DeletionTimestamp != nil {
		return nil, projectHelmChartStatus, nil
	}

	// handle charts with cleanup label
	if common2.HasCleanupLabel(projectHelmChart) {
		projectHelmChartStatus = h.getCleanupStatus(projectHelmChart, projectHelmChartStatus)
		logrus.Infof("Cleaning up HelmChart and HelmRelease for ProjectHelmChart %s/%s", projectHelmChart.Namespace, projectHelmChart.Name)
		return nil, projectHelmChartStatus, nil
	}

	// get information about the projectHelmChart
	projectID, err := h.getProjectID(projectHelmChart)
	if err != nil {
		return nil, projectHelmChartStatus, err
	}
	releaseNamespace, releaseName := h.getReleaseNamespaceAndName(projectHelmChart)

	// check if the releaseName is already tracked by another ProjectHelmChart
	projectHelmCharts, err := h.projectHelmChartCache.GetByIndex(ProjectHelmChartByReleaseName, releaseName)
	if err != nil {
		return nil, projectHelmChartStatus, fmt.Errorf("unable to get ProjectHelmCharts to verify if release is already tracked: %s", err)
	}
	for _, conflictingProjectHelmChart := range projectHelmCharts {
		if conflictingProjectHelmChart == nil {
			continue
		}
		if projectHelmChart.Name == conflictingProjectHelmChart.Name && projectHelmChart.Namespace == conflictingProjectHelmChart.Namespace {
			// looking at the same projectHelmChart that we have at hand
			continue
		}
		if len(conflictingProjectHelmChart.Status.Status) == 0 {
			// the other ProjectHelmChart hasn't been processed yet, so let it fail out whenever it is processed
			continue
		}
		if conflictingProjectHelmChart.Status.Status == "UnableToCreateHelmRelease" {
			// the other ProjectHelmChart is the one that will not be able to progress, so we can continue to update this one
			continue
		}
		// we have found another ProjectHelmChart that already exists and is tracking this release with some non-conflicting status
		err = fmt.Errorf(
			"ProjectHelmChart %s/%s already tracks release %s/%s",
			conflictingProjectHelmChart.Namespace, conflictingProjectHelmChart.Name,
			releaseName, releaseNamespace,
		)
		projectHelmChartStatus = h.getUnableToCreateHelmReleaseStatus(projectHelmChart, projectHelmChartStatus, err)
		return nil, projectHelmChartStatus, nil
	}

	// set basic statuses
	projectHelmChartStatus.SystemNamespace = h.systemNamespace
	projectHelmChartStatus.ReleaseNamespace = releaseNamespace
	projectHelmChartStatus.ReleaseName = releaseName

	// gather target project namespaces
	targetProjectNamespaces, err := h.projectGetter.GetTargetProjectNamespaces(projectHelmChart)
	if err != nil {
		return nil, projectHelmChartStatus, fmt.Errorf("unable to find project namespaces to deploy ProjectHelmChart: %s", err)
	}
	if len(targetProjectNamespaces) == 0 {
		projectReleaseNamespace := h.getProjectReleaseNamespace(projectID, true, projectHelmChart)
		if projectReleaseNamespace != nil {
			objs = append(objs, projectReleaseNamespace)
		}
		projectHelmChartStatus = h.getNoTargetNamespacesStatus(projectHelmChart, projectHelmChartStatus)
		return objs, projectHelmChartStatus, nil
	}

	if releaseNamespace != h.systemNamespace && releaseNamespace != projectHelmChart.Namespace {
		// need to add release namespace to list of objects to be created
		projectReleaseNamespace := h.getProjectReleaseNamespace(projectID, false, projectHelmChart)
		objs = append(objs, projectReleaseNamespace)
		// need to add auto-generated release namespace to target namespaces
		targetProjectNamespaces = append(targetProjectNamespaces, releaseNamespace)
	}
	projectHelmChartStatus.TargetNamespaces = targetProjectNamespaces

	// get values.yaml from ProjectHelmChart spec and default overrides
	values := h.getValues(projectHelmChart, projectID, targetProjectNamespaces)
	valuesContentBytes, err := values.ToYAML()
	if err != nil {
		err = fmt.Errorf("unable to marshall spec.values: %s", err)
		projectHelmChartStatus = h.getValuesParseErrorStatus(projectHelmChart, projectHelmChartStatus, err)
		return nil, projectHelmChartStatus, nil
	}

	ns, err := h.namespaceCache.Get(releaseNamespace)
	if ns == nil || apierrors.IsNotFound(err) {
		// The release namespace does not exist yet, create it and leave the status as UnableToCreateHelmRelease
		//
		// Note: since we have a resolver that watches for the project release namespace, this handler will get re-enqueued
		//
		// Note: the reason why we need to do this check is to ensure that deleting a project release namespace will delete
		// and recreate the HelmChart and HelmRelease resources, which will ensure that the HelmChart gets re-installed onto
		// the newly created namespace. Without this, a deleted release namespace will always have ProjectHelmCharts stuck in
		// WaitingForDashboardValues since the underlying helm release will never be recreated
		err = fmt.Errorf("cannot find release namespace %s to deploy release", releaseNamespace)
		projectHelmChartStatus = h.getUnableToCreateHelmReleaseStatus(projectHelmChart, projectHelmChartStatus, err)
		return objs, projectHelmChartStatus, nil
	} else if err != nil {
		return nil, projectHelmChartStatus, err
	}

	// get rolebindings that need to be created in release namespace
	k8sRolesToRoleRefs, err := h.getSubjectRoleToRoleRefsFromRoles(projectHelmChart)
	if err != nil {
		return nil, projectHelmChartStatus, fmt.Errorf("unable to get release roles from project release namespace %s for %s/%s: %s", releaseNamespace, projectHelmChart.Namespace, projectHelmChart.Name, err)
	}
	k8sRolesToSubjects, err := h.getSubjectRoleToSubjectsFromBindings(projectHelmChart)
	if err != nil {
		return nil, projectHelmChartStatus, fmt.Errorf("unable to get rolebindings to default project operator roles from project registration namespace %s for %s/%s: %s", projectHelmChart.Namespace, projectHelmChart.Namespace, projectHelmChart.Name, err)
	}
	objs = append(objs,
		h.getRoleBindings(projectID, k8sRolesToRoleRefs, k8sRolesToSubjects, projectHelmChart)...,
	)

	// append the helm chart and helm release
	objs = append(objs,
		h.getHelmChart(projectID, string(valuesContentBytes), projectHelmChart),
		h.getHelmRelease(projectID, projectHelmChart),
	)

	// get dashboard values if available
	dashboardValues, err := h.getDashboardValuesFromConfigmaps(projectHelmChart)
	if err != nil {
		return nil, projectHelmChartStatus, fmt.Errorf("unable to get dashboard values from status ConfigMaps: %s", err)
	}
	if len(dashboardValues) == 0 {
		projectHelmChartStatus = h.getWaitingForDashboardValuesStatus(projectHelmChart, projectHelmChartStatus)
	} else {
		projectHelmChartStatus.DashboardValues = dashboardValues
		projectHelmChartStatus = h.getDeployedStatus(projectHelmChart, projectHelmChartStatus)
	}
	return objs, projectHelmChartStatus, nil
}

func (h *handler) OnRemove(_ string, projectHelmChart *v1alpha2.ProjectHelmChart) (*v1alpha2.ProjectHelmChart, error) {
	if projectHelmChart == nil {
		return nil, nil
	}

	// get information about the projectHelmChart
	projectID, err := h.getProjectID(projectHelmChart)
	if err != nil {
		return projectHelmChart, err
	}

	// Get orphaned release namespace and apply it; if another ProjectHelmChart exists in this namespace, it will automatically remove
	// the orphaned label on enqueuing the namespace since that will enqueue all ProjectHelmCharts associated with it
	projectReleaseNamespace := h.getProjectReleaseNamespace(projectID, true, projectHelmChart)
	if projectReleaseNamespace == nil {
		// nothing to be done since this operator does not create project release namespaces
		return projectHelmChart, nil
	}

	// Why aren't we modifying the set ID or owner here?
	// Since this applier runs without deleting objects whose GVKs indicate that they are namespaces,
	// we don't have to worry about another controller using this same set ID (e.g. another Project Operator)
	// that will delete this projectReleaseNamespace on seeing it
	err = h.apply.ApplyObjects(projectReleaseNamespace)
	if err != nil {
		return projectHelmChart, fmt.Errorf("unable to add orphaned annotation to project release namespace %s", projectReleaseNamespace.Name)
	}
	return projectHelmChart, nil
}
