package common

import (
	rbacv1 "k8s.io/api/rbac/v1"
)

// GetDefaultClusterRoles returns the default ClusterRoles that this operator was started with
func GetDefaultClusterRoles(opts Options) map[string]string {
	clusterRoles := make(map[string]string)
	if len(opts.AdminClusterRole) > 0 {
		clusterRoles["admin"] = opts.AdminClusterRole
	}
	if len(opts.EditClusterRole) > 0 {
		clusterRoles["edit"] = opts.EditClusterRole
	}
	if len(opts.ViewClusterRole) > 0 {
		clusterRoles["view"] = opts.ViewClusterRole
	}
	return clusterRoles
}

// IsDefaultClusterRoleRef returns whether the provided name is a default ClusterRole ref that this operator was
// started with (e.g. the values provided to AdminClusterRole, EditClusterRole, or ViewClusterRole in RuntimeOptions)
func IsDefaultClusterRoleRef(opts Options, roleRefName string) (string, bool) {
	for subjectRole, defaultClusterRoleName := range GetDefaultClusterRoles(opts) {
		if roleRefName == defaultClusterRoleName {
			return subjectRole, true
		}
	}
	return "", false
}

// FilterToUsersAndGroups returns a subset of the provided subjects that are only Users and Groups
// i.e. it filters out ServiceAccount subjects
func FilterToUsersAndGroups(subjects []rbacv1.Subject) []rbacv1.Subject {
	var filtered []rbacv1.Subject
	for _, subject := range subjects {
		if subject.APIGroup != rbacv1.GroupName {
			continue
		}
		if subject.Kind != rbacv1.UserKind && subject.Kind != rbacv1.GroupKind {
			// we do not automatically bind service accounts, only users and groups
			continue
		}
		// note: we are purposefully omitting namespace here since it is not necessary even if set
		filtered = append(filtered, rbacv1.Subject{
			APIGroup: subject.APIGroup,
			Kind:     subject.Kind,
			Name:     subject.Name,
		})
	}
	return filtered
}
