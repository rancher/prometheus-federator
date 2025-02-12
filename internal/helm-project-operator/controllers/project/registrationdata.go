package project

import (
	"fmt"
	"time"

	"github.com/rancher/prometheus-federator/internal/helm-project-operator/apis/helm.cattle.io/v1alpha1"
	"github.com/rancher/prometheus-federator/internal/helm-project-operator/controllers/common"

	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)


const (
	// Default values for backoff
	defaultRetryTimeout  = 30 * time.Second
	maxRetries = 5
	defaultJitter = 0.1
)

// Note: each resource created here should have a resolver set in resolvers.go

// getSubjectRoleToSubjectsFromBindings gets all RoleBindings in the Project Registration Namespace that need to be synced to assign the corresponding
// permission in the Project Release Namespace. See pkg/controllers/project/resources.go for more information on how this is used
func (h *handler) getSubjectRoleToSubjectsFromBindings(projectHelmChart *v1alpha1.ProjectHelmChart) (map[string][]rbacv1.Subject, error) {
	defaultClusterRoles := common.GetDefaultClusterRoles(h.opts)
	subjectRoleToSubjects := make(map[string][]rbacv1.Subject)
	subjectRoleToSubjectMap := make(map[string]map[string]rbacv1.Subject)
	if len(defaultClusterRoles) == 0 {
		// no roles to get get subjects for
		return subjectRoleToSubjects, nil
	}
	for subjectRole := range defaultClusterRoles {
		subjectRoleToSubjectMap[subjectRole] = make(map[string]rbacv1.Subject)
	}
	
	// fetch the list of roleBindings to be created
	roleBindings, err := h.fetchRoleBindings(projectHelmChart.Namespace)
	if err != nil{
		return nil, fmt.Errorf("failed to process role bindings: %w", err)
	}
	for _, rb := range roleBindings {
		if rb == nil {
			continue
		}
		subjectRole, isDefaultRoleRef := common.IsDefaultClusterRoleRef(h.opts, rb.RoleRef.Name)
		if !isDefaultRoleRef {
			logrus.Debugf("Role %s is not a default role for %s", subjectRole, projectHelmChart.Namespace)
			continue
		}
		filteredSubjects := common.FilterToUsersAndGroups(rb.Subjects)
		currSubjects := subjectRoleToSubjectMap[subjectRole]
		for _, filteredSubject := range filteredSubjects {
			// collect into a map to avoid putting duplicates of the same subject
			// we use an index of kind and name since a Group can have the same name as a User, but should be considered separate
			currSubjects[fmt.Sprintf("%s-%s", filteredSubject.Kind, filteredSubject.Name)] = filteredSubject
		}
	}
	
	// fetch the list of clusterRoleBindings to be created
	clusterRoleBindings, err := h.fetchClusterRoleBindings()
	if err != nil{
		return nil, fmt.Errorf("failed to process cluster role bindings: %w", err)
	}
	for _, crb := range clusterRoleBindings {
		if crb == nil {
			continue
		}
		subjectRole, isDefaultRoleRef := common.IsDefaultClusterRoleRef(h.opts, crb.RoleRef.Name)
		if !isDefaultRoleRef {
			continue
		}
		filteredSubjects := common.FilterToUsersAndGroups(crb.Subjects)
		currSubjects := subjectRoleToSubjectMap[subjectRole]
		for _, filteredSubject := range filteredSubjects {
			// collect into a map to avoid putting duplicates of the same subject
			// we use an index of kind and name since a Group can have the same name as a User, but should be considered separate
			currSubjects[fmt.Sprintf("%s-%s", filteredSubject.Kind, filteredSubject.Name)] = filteredSubject
		}
	}
	
	// convert back into list so that no duplicates are created
	for subjectRole := range defaultClusterRoles {
		subjects := []rbacv1.Subject{}
		for _, subject := range subjectRoleToSubjectMap[subjectRole] {
			subjects = append(subjects, subject)
		}
		subjectRoleToSubjects[subjectRole] = subjects
	}

	return subjectRoleToSubjects, nil
}

// fetchResourceWithRetry attempts to fetch a resource with an exponential backoff retry strategy.
func (h *handler) fetchResourceWithRetry(resourceType string, fetchFunc func() (bool, error)) error{
	backoffPolicy := wait.Backoff{
		Duration: defaultRetryTimeout,
		Jitter:   defaultJitter,
		Steps:    maxRetries,
	}
	
	err := wait.ExponentialBackoff(backoffPolicy, fetchFunc)
	if err != nil {
		return fmt.Errorf("failed to fetch %s after %v retries: %v", resourceType, backoffPolicy.Steps, err)
	}
	
	return nil
}

// fetchRoleBindings retrieves RoleBindings for a given namespace.
// It first attempts to fetch them from the cache; if the cache is empty, it queries the API server.
// The function uses an exponential backoff retry strategy to handle transient failures.
func (h *handler) fetchRoleBindings(namespace string) ([]*rbacv1.RoleBinding, error){
	var roleBindings []*rbacv1.RoleBinding
	var err error
	
	fetchFunc := func() (bool, error){
		// fetch the roleBindings from cache
		roleBindings, err = h.rolebindingCache.GetByIndex(
			RoleBindingInRegistrationNamespaceByRoleRef,
			NamespacedBindingReferencesDefaultOperatorRole(namespace),
		)
				
		// if cache returns empty list of roleBinding, query the API server
		if err == nil && len(roleBindings) == 0{
			logrus.Debug("RoleBinding cache returned empty results, attempting direct API server query")
			roleBindingList, err := h.rolebindings.List(namespace, metav1.ListOptions{})
			if err != nil{
				return false, fmt.Errorf("failed to fetch rolebindings from API server: %w", err)
			}
						
			for idx := range roleBindingList.Items{
				roleBindings = append(roleBindings, &roleBindingList.Items[idx])
			}
		} else if err != nil {
			return false, fmt.Errorf("failed to fetch rolebindings from cache: %w", err)
		} 
		
		return true, nil
	}
	
	if err := h.fetchResourceWithRetry("roleBindings", fetchFunc); err != nil{
		return nil, err
	}
	
	return roleBindings, nil
}

// fetchClusterRoleBindings retrieves required ClusterRoleBindings.
// It first attempts to fetch them from the cache; if the cache is empty, it queries the API server.
// The function uses an exponential backoff retry strategy to handle transient failures.
func (h *handler) fetchClusterRoleBindings() ([]*rbacv1.ClusterRoleBinding, error){
	var clusterRoleBindings []*rbacv1.ClusterRoleBinding
	var err error
	
	fetchFunc := func() (bool, error){
		// fetch the clusterRoleBindings from cache
		clusterRoleBindings, err = h.clusterrolebindingCache.GetByIndex(
			ClusterRoleBindingByRoleRef, BindingReferencesDefaultOperatorRole)
				
		// if cache returns empty list of clusterRoleBinding, query the API server
		if err == nil && len(clusterRoleBindings) == 0{
			logrus.Debug("ClusterRoleBinding cache returned empty results, attempting direct API server query")
			clusterRoleBindingList, err := h.clusterrolebindings.List(metav1.ListOptions{})
			if err != nil{
				return false, fmt.Errorf("failed to fetch ClusterRolebindings from API server: %w", err)
			}
						
			for idx := range clusterRoleBindingList.Items{
				clusterRoleBindings = append(clusterRoleBindings, &clusterRoleBindingList.Items[idx])
			}
		} else if err != nil {
			return false, fmt.Errorf("failed to fetch clusterRolebindings from cache: %w", err)
		} 
		
		return true, nil
	}
	
	if err := h.fetchResourceWithRetry("clusterRoleBindings", fetchFunc); err != nil{
		return nil, err
	}
	
	return clusterRoleBindings, nil
}