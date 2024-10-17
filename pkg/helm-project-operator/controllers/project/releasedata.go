package project

import (
	"encoding/json"
	"fmt"
	"strings"

	v1alpha2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	common2 "github.com/rancher/prometheus-federator/pkg/helm-project-operator/controllers/common"

	"github.com/rancher/wrangler/pkg/data"
	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Note: each resource created here should have a resolver set in resolvers.go

// getDashboardValuesFromConfigMaps returns the generic map that represents a merge of all the contents of all ConfigMaps in the
// Project Release Namespace with the label helm.cattle.io/dashboard-values-configmap: {{ .Release.Name }}.
//
// Generally, these ConfigMaps should be part of the deployed Helm chart and should not have conflicts with each other
// It's also a common pattern to only have a single ConfigMap that this refers to.
func (h *handler) getDashboardValuesFromConfigmaps(projectHelmChart *v1alpha2.ProjectHelmChart) (v1alpha2.GenericMap, error) {
	releaseNamespace, releaseName := h.getReleaseNamespaceAndName(projectHelmChart)
	exists, err := h.verifyReleaseNamespaceExists(releaseNamespace)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	configMaps, err := h.configmapCache.GetByIndex(ConfigMapInReleaseNamespaceByReleaseNamespaceName, fmt.Sprintf("%s/%s", releaseNamespace, releaseName))
	if err != nil {
		return nil, err
	}
	var values v1alpha2.GenericMap
	for _, configMap := range configMaps {
		if configMap == nil {
			continue
		}
		for jsonKey, jsonContent := range configMap.Data {
			if !strings.HasSuffix(jsonKey, ".json") {
				logrus.Errorf("dashboard values configmap %s/%s has non-JSON key %s, expected only keys ending with .json. skipping...", configMap.Namespace, configMap.Name, jsonKey)
				continue
			}
			var jsonMap map[string]interface{}
			err := json.Unmarshal([]byte(jsonContent), &jsonMap)
			if err != nil {
				logrus.Errorf("could not marshall content in dashboard values configmap %s/%s in key %s (err='%s'). skipping...", configMap.Namespace, configMap.Name, jsonKey, err)
				continue
			}
			values = data.MergeMapsConcatSlice(values, jsonMap)
		}
	}
	return values, nil
}

// getSubjectRoleToRoleRefsFromRoles gets all Roles in the Project Release Namespace that need RoleBindings to be created automatically
// based on permissions set in the Project Registration namespace. See pkg/controllers/project/resources.go for more information on how this is used
func (h *handler) getSubjectRoleToRoleRefsFromRoles(projectHelmChart *v1alpha2.ProjectHelmChart) (map[string][]rbacv1.RoleRef, error) {
	subjectRoleToRoleRefs := make(map[string][]rbacv1.RoleRef)
	for subjectRole := range common2.GetDefaultClusterRoles(h.opts) {
		subjectRoleToRoleRefs[subjectRole] = []rbacv1.RoleRef{}
	}
	if len(subjectRoleToRoleRefs) == 0 {
		// no roles were defined to be auto-aggregated
		return subjectRoleToRoleRefs, nil
	}
	releaseNamespace, releaseName := h.getReleaseNamespaceAndName(projectHelmChart)
	exists, err := h.verifyReleaseNamespaceExists(releaseNamespace)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	roles, err := h.roleCache.GetByIndex(RoleInReleaseNamespaceByReleaseNamespaceName, fmt.Sprintf("%s/%s", releaseNamespace, releaseName))
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role == nil {
			continue
		}
		subjectRole, ok := role.Labels[common2.HelmProjectOperatorProjectHelmChartRoleAggregateFromLabel]
		if !ok {
			// cannot assign roles if this label is not provided
			continue
		}
		roleRefs, ok := subjectRoleToRoleRefs[subjectRole]
		if !ok {
			// label value is invalid since it does not point to default subject role name
			continue
		}
		subjectRoleToRoleRefs[subjectRole] = append(roleRefs, rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     role.Name,
		})
	}
	return subjectRoleToRoleRefs, nil
}

func (h *handler) verifyReleaseNamespaceExists(releaseNamespace string) (bool, error) {
	_, err := h.namespaceCache.Get(releaseNamespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// release namespace has not been created yet
			return false, nil
		}
		return false, err
	}
	return true, nil
}
