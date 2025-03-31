package namespace

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/rancher/prometheus-federator/internal/helm-project-operator/applier"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"
	helmprojectcontroller "github.com/rancher/prometheus-federator/internal/helm-project-operator/generated/controllers/helm.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/apply"
	corecontroller "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"
)

type handler struct {
	namespaceApply apply.Apply
	apply          apply.Apply

	systemNamespace string
	valuesYaml      string
	questionsYaml   string
	opts            common.Options

	systemNamespaceTracker                         Tracker
	projectRegistrationNamespaceTracker            Tracker
	projectRegistrationNamespaceInitializationList []string

	namespaces            corecontroller.NamespaceController
	namespaceCache        corecontroller.NamespaceCache
	configmaps            corecontroller.ConfigMapController
	projectHelmCharts     helmprojectcontroller.ProjectHelmChartController
	projectHelmChartCache helmprojectcontroller.ProjectHelmChartCache

	projectRegistrationNamespaceApplyinator applier.Applyinator
}

func Register(
	ctx context.Context,
	apply apply.Apply,
	systemNamespace, valuesYaml, questionsYaml string,
	opts common.Options,
	namespaces corecontroller.NamespaceController,
	namespaceCache corecontroller.NamespaceCache,
	configmaps corecontroller.ConfigMapController,
	projectHelmCharts helmprojectcontroller.ProjectHelmChartController,
	projectHelmChartCache helmprojectcontroller.ProjectHelmChartCache,
	dynamic dynamic.Interface,
) ProjectGetter {

	apply = apply.WithCacheTypes(configmaps)

	h := &handler{
		apply:                               apply,
		systemNamespace:                     systemNamespace,
		valuesYaml:                          valuesYaml,
		questionsYaml:                       questionsYaml,
		opts:                                opts,
		systemNamespaceTracker:              NewTracker(),
		projectRegistrationNamespaceTracker: NewTracker(),
		projectRegistrationNamespaceInitializationList: []string{},
		namespaces:            namespaces,
		namespaceCache:        namespaceCache,
		configmaps:            configmaps,
		projectHelmCharts:     projectHelmCharts,
		projectHelmChartCache: projectHelmChartCache,
	}

	// note: this implements a workqueue that ensures that applies only happen once at a time even if a bunch of namespaces in a project
	// are all re-enqueued at the exact same time
	h.projectRegistrationNamespaceApplyinator = applier.NewApplyinator("project-registration-namespace-applyinator", h.applyProjectRegistrationNamespace, nil)
	h.projectRegistrationNamespaceApplyinator.Run(ctx, opts.NamespaceRegistrationWorkers)
	logrus.Debugf("Initializing namespace applyinator with %d namespace registration workers", opts.NamespaceRegistrationWorkers)

	h.apply = h.addReconcilers(h.apply, dynamic)

	h.initResolvers(ctx)

	h.initIndexers()

	if len(opts.ProjectLabel) == 0 {
		namespaces.OnChange(ctx, "on-namespace-change", h.OnSingleNamespaceChange)

		return NewSingleNamespaceProjectGetter(systemNamespace, opts.SystemNamespaces, namespaces)
	}

	// the namespaceApply is only needed in a multi-namespace setup
	// note: we never delete namespaces that are created since it's possible that the user may want to leave them around
	// on remove, we only output a log that says that the user should clean it up and add an annotation that it is orphaned
	h.namespaceApply = apply.
		WithSetID("project-registration-namespace-applier").
		WithCacheTypes(namespaces).
		WithNoDeleteGVK(namespaces.GroupVersionKind())

	namespaces.OnChange(ctx, "on-namespace-change", h.OnMultiNamespaceChange)

	h.initSystemNamespaces(h.opts.SystemNamespaces, h.systemNamespaceTracker)

	err := h.initProjectRegistrationNamespaces()
	if err != nil {
		logrus.Fatal(err)
	}

	compareNamespaceErr := retry.OnError(
		wait.Backoff{
			Steps:    opts.NamespaceRegistrationRetryMax,
			Duration: time.Duration(opts.NamespaceRegistrationRetryWaitMilliseconds) * time.Millisecond,
			Factor:   1.0,
		},
		func(_ error) bool {
			logrus.Warnf("Registered Project Registration namespaces don't match expected, will retry. Registered: %v. Expected: %v.", h.projectRegistrationNamespaceTracker.List(), h.projectRegistrationNamespaceInitializationList)
			return true
		}, func() error {
			return h.projectRegistrationNamespaceTracker.Compare(h.projectRegistrationNamespaceInitializationList)
		})
	if compareNamespaceErr != nil {
		logrus.Fatal("The operator has failed registering all project registration namespace into its Tracker in a timely manner. Please bump the number of namespace registration workers or increase the number of retries.")
	}

	logrus.Infof("Initializing namespace controller with the following namespaces tracked as Project Registration namespaces: %v", h.projectRegistrationNamespaceTracker.List())

	return NewLabelBasedProjectGetter(h.opts.ProjectLabel, h.isProjectRegistrationNamespace, h.isSystemNamespace, h.namespaces)
}

// Single Namespace Handler

func (h *handler) OnSingleNamespaceChange( /*name*/ _ string, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	if namespace.Name != h.systemNamespace {
		// enqueue system namespace to ensure that rolebindings are updated

		logrus.Debugf("Enqueue system namespace to ensure that rolebindings are updated in OnSingleNamespaceChange: %s", h.systemNamespace)
		h.namespaces.Enqueue(h.systemNamespace)
		return namespace, nil
	}
	if namespace.DeletionTimestamp != nil {
		// When a namespace gets deleted, the ConfigMap deployed in that namespace should also get deleted
		// Therefore, we do not need to apply anything in this situation to avoid spamming logs with trying to apply
		// a resource to a namespace that is being terminated
		logrus.Debugf("OnSingleNamespaceChange %s has deletion timestamp of %v", namespace, namespace.DeletionTimestamp)
		return namespace, nil
	}
	// Trigger applying the data for this projectRegistrationNamespace
	var objs []runtime.Object
	objs = append(objs, h.getConfigMap("", namespace))
	return namespace, h.configureApplyForNamespace(namespace).ApplyObjects(objs...)
}

// Multiple Namespaces Handler

func (h *handler) OnMultiNamespaceChange( /*name*/ _ string, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	if namespace == nil {
		logrus.Debugf("OnMultiNamespaceChange() called with no namespace.")
		return namespace, nil
	}

	switch {
	// note: the check for a project registration namespace must happen before
	// we check for whether it is a system namespace to address the scenario where
	// the 'projectLabel: systemProjectLabelValue' is added to the project registration
	// namespace, which will cause it to be ignored and left in the System Project unless
	// we apply the ProjectRegistrationNamespace logic first.
	case h.isProjectRegistrationNamespace(namespace):
		logrus.Debugf("Detected \"%s\" as a project registration namespace. Enqueueing it.", namespace.Name)
		err := h.enqueueProjectNamespaces(namespace)
		if err != nil {
			logrus.Debugf("Error in call to isProjectRegistrationNamespace() while enqueuing project namespace %s: %s", namespace, err)
			return namespace, err
		}
		if namespace.DeletionTimestamp != nil {
			logrus.Debugf("%s has deletion timestamp %v in isProjectRegistrationNamespace()", namespace, namespace.DeletionTimestamp)
			h.projectRegistrationNamespaceTracker.Delete(namespace)
		}
		return namespace, nil
	case h.isSystemNamespace(namespace):
		logrus.Infof("Detected \"%s\" as a system namespace. Ignoring it.", namespace.Name)
		// nothing to do, we always ignore system namespaces
		return namespace, nil
	default:
		logrus.Infof("Detected \"%s\" as a regular namespace. Handling it", namespace.Name)
		err := h.applyProjectRegistrationNamespaceForNamespace(namespace)
		if err != nil {
			logrus.Debugf("Default error in isProjectRegistrationNamespace() %s: %s", namespace, err)
			return namespace, err
		}
		return namespace, nil
	}
}

func (h *handler) enqueueProjectNamespaces(projectRegistrationNamespace *corev1.Namespace) error {
	if projectRegistrationNamespace == nil {
		return nil
	}
	// ensure that we are working with the projectRegistrationNamespace that we expect, not the one we found
	expectedNamespace, exists := h.projectRegistrationNamespaceTracker.Get(projectRegistrationNamespace.Name)
	if !exists {
		// we no longer expect this namespace to exist, so don't enqueue any namespaces
		return nil
	}
	// projectRegistrationNamespace was modified or removed, so we should re-enqueue any namespaces tied to it
	projectID, ok := expectedNamespace.Labels[h.opts.ProjectLabel]
	if !ok {
		return fmt.Errorf("could not find project that projectRegistrationNamespace %s is tied to", projectRegistrationNamespace.Name)
	}
	projectNamespaces, err := h.namespaceCache.GetByIndex(NamespacesByProjectExcludingRegistrationID, projectID)
	if err != nil {
		return err
	}
	for _, ns := range projectNamespaces {
		h.namespaces.Enqueue(ns.Name)
	}
	logrus.Debugf("ProjectRegistrationNamespace %s was modified or removed in call to enqueueProjectNamespaces(). Reenqueiing any namepsaced tied to it.", projectRegistrationNamespace.Name)
	return nil
}

func (h *handler) applyProjectRegistrationNamespaceForNamespace(namespace *corev1.Namespace) error {
	// get the project ID and generate the namespace object to be applied
	projectID, inProject := h.getProjectIDFromNamespaceLabels(namespace)
	logrus.Debugf("Found namespace %s to be part of the project %s. inProject = %v. Labels will be updated accordingly.", namespace.Name, projectID, inProject)

	// update the namespace with the appropriate label on it
	err := h.updateNamespaceWithHelmOperatorProjectLabel(namespace, projectID, inProject)
	if err != nil {
		logrus.Debugf("Error updating namespace %s with %s labels", namespace, projectID)
		return nil
	}
	if !inProject {
		return nil
	}

	logrus.Infof("Calling projectRegistrationNamespaceApplyinator for project %s", projectID)
	// Note: why do we use an Applyinator.Apply here instead of just directly
	// running h.applyProjectRegistrationNamespace?
	//
	// If we ran the logic for applying a Project Registration Namespace here,
	// on every time a Project Namespace was re-enqueued, that would result in projects
	// with a lot of namespaces all trying to run the exact same apply operation
	// at the exact same time; however, the client-go workqueue implementation
	// (which lasso controllers use under the hood as well) allow us to add the registration
	// namespace to the queue with certain guarantees, namely this one that we need:
	//
	// * Stingy: a single item will not be processed multiple times concurrently,
	// and if an item is added multiple times before it can be processed, it
	// will only be processed once.
	//
	// This ensures that the actual application of a project registration namespace
	// will only happen once, regardless of how many enqueues, which prevents us
	// from hammering wrangler.Apply operations and forcing wrangler.Apply to engage
	// in rate limiting (and output noisy logs)

	// Every namespace named as 'cattle-project-projectID' will be handled as a Project Registration
	// namespace and added to a list that synchronously tracks every project
	// registration namespace name.

	// An extra synchronous list is important because
	// 'h.projectRegistrationNamespaceApplyinator.Apply(projectID)' will add to an asynchronous workqueue
	// and the controller has to wait until all of them have been registered to initialize properly.
	// Thus, the synchronous list works as a way to compare which namespaces should be found
	// in the asynchronous tracker and prevent controller initialization
	// until both match.

	// The asynchronous tracker has to be kept asynchronous because it stores
	// not only the names, but the current state of the namespaces
	// as well. So they have to be updated before getting added to
	// the tracker
	projectRegistrationNamespace := h.getProjectRegistrationNamespaceName(projectID)
	if !slices.Contains(h.projectRegistrationNamespaceInitializationList, projectRegistrationNamespace) {
		logrus.Debugf("Triggered by namespace %s", namespace.Name)
		logrus.Debugf("Project %s found to have %s as its project registration namespace. %s will be synchronously added to the projectRegistrationNamespaceInitializationList", projectID, projectRegistrationNamespace, projectRegistrationNamespace)
		h.projectRegistrationNamespaceInitializationList = append(h.projectRegistrationNamespaceInitializationList, projectRegistrationNamespace)
	}

	h.projectRegistrationNamespaceApplyinator.Apply(projectID)

	return nil
}

func (h *handler) applyProjectRegistrationNamespace(projectID string) error {
	// Calculate whether to add the orphaned label
	var isOrphaned bool
	projectNamespaces, err := h.namespaceCache.GetByIndex(NamespacesByProjectExcludingRegistrationID, projectID)
	if err != nil {
		return err
	}
	var numNamespaces int
	for _, ns := range projectNamespaces {
		if ns.DeletionTimestamp != nil {
			// ignore namespaces that are being deleted
			continue
		}
		numNamespaces++
	}
	if numNamespaces == 0 {
		// add orphaned label and trigger a warning
		isOrphaned = true
	}
	logrus.Debugf("Calculated 'isOrphaned' to %v for %s", isOrphaned, projectID)

	// get the resources and validate them
	projectRegistrationNamespace := h.getProjectRegistrationNamespace(projectID, isOrphaned)
	logrus.Debugf("Got project registration namespace '%s' for project %s", projectRegistrationNamespace.Name, projectID)
	// ensure that the projectRegistrationNamespace created from this projectID is valid
	if len(projectRegistrationNamespace.Name) > 63 {
		// ensure that we don't try to create a namespace with too big of a name
		logrus.Errorf("could not apply namespace with name %s: name is above 63 characters", projectRegistrationNamespace.Name)
		return nil
	}

	// Trigger the apply and set the projectRegistrationNamespace
	err = h.namespaceApply.ApplyObjects(projectRegistrationNamespace)
	if err != nil {
		return err
	}

	// get the projectRegistrationNamespace after applying to get a valid object to pass in as the owner of the next apply
	projectRegistrationNamespace, err = h.namespaces.Get(projectRegistrationNamespace.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get project registration namespace from cache after create: %s", err)
	}
	logrus.Debugf("Adding namespace %s to the global Project Registration namespace tracker", projectRegistrationNamespace.Name)
	h.projectRegistrationNamespaceTracker.Set(projectRegistrationNamespace)

	if projectRegistrationNamespace.DeletionTimestamp != nil {
		// When a namespace gets deleted, the ConfigMap deployed in that namespace and all ProjectHelmCharts should also get deleted
		// Therefore, we do not need to apply anything in this situation to avoid spamming logs with trying to apply
		// a resource to a namespace that is being terminated
		//
		// We expect this to be recalled when the project registration namespace is recreated anyways
		return nil
	}

	// Trigger applying the data for this projectRegistrationNamespace
	var objs []runtime.Object
	objs = append(objs, h.getConfigMap(projectID, projectRegistrationNamespace))
	err = h.configureApplyForNamespace(projectRegistrationNamespace).ApplyObjects(objs...)
	if err != nil {
		return err
	}

	// ensure that all ProjectHelmCharts are re-enqueued within this projectRegistrationNamespace
	err = h.enqueueProjectHelmChartsForNamespace(projectRegistrationNamespace)
	if err != nil {
		return fmt.Errorf("unable to re-enqueue ProjectHelmCharts on reconciling change to namespaces in project %s: %s", projectID, err)
	}

	return nil
}

func (h *handler) updateNamespaceWithHelmOperatorProjectLabel(namespace *corev1.Namespace, projectID string, inProject bool) error {
	if namespace.DeletionTimestamp != nil {
		// no need to update a namespace about to be deleted
		return nil
	}
	if len(h.opts.ProjectReleaseLabelValue) == 0 {
		// do nothing, this annotation is irrelevant unless we create release namespaces
		return nil
	}
	if len(projectID) == 0 || !inProject {
		// ensure that the HelmProjectOperatorProjectLabel is removed if added
		if namespace.Labels == nil {
			return nil
		}
		if _, ok := namespace.Labels[common.HelmProjectOperatorProjectLabel]; !ok {
			return nil
		}
		namespaceCopy := namespace.DeepCopy()
		delete(namespaceCopy.Labels, common.HelmProjectOperatorProjectLabel)
		_, err := h.namespaces.Update(namespaceCopy)
		if err != nil {
			return err
		}
	}

	namespaceCopy := namespace.DeepCopy()
	if namespaceCopy.Labels == nil {
		namespaceCopy.Labels = map[string]string{}
	}
	currLabel, ok := namespaceCopy.Labels[common.HelmProjectOperatorProjectLabel]
	if !ok || currLabel != projectID {
		namespaceCopy.Labels[common.HelmProjectOperatorProjectLabel] = projectID
	}
	_, err := h.namespaces.Update(namespaceCopy)
	if err != nil {
		return err
	}
	return nil
}

func (h *handler) isProjectRegistrationNamespace(namespace *corev1.Namespace) bool {
	if namespace == nil {
		return false
	}
	return h.projectRegistrationNamespaceTracker.Has(namespace.Name)
}

func (h *handler) isSystemNamespace(namespace *corev1.Namespace) bool {
	if namespace == nil {
		return false
	}
	isTrackedSystemNamespace := h.systemNamespaceTracker.Has(namespace.Name)
	if isTrackedSystemNamespace {
		return true
	}

	var systemProjectLabelValues []string
	if len(h.opts.SystemProjectLabelValues) != 0 {
		systemProjectLabelValues = append(systemProjectLabelValues, h.opts.SystemProjectLabelValues...)
	}
	if len(h.opts.ProjectReleaseLabelValue) != 0 {
		systemProjectLabelValues = append(systemProjectLabelValues, h.opts.ProjectReleaseLabelValue)
	}
	projectID, inProject := h.getProjectIDFromNamespaceLabels(namespace)
	if !inProject {
		return false
	}
	for _, systemProjectLabelValue := range systemProjectLabelValues {
		// check if labels indicate this is a system project
		if projectID == systemProjectLabelValue {
			return true
		}
	}
	return false
}
