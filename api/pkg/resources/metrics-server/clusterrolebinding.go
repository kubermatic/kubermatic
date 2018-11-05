package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingResourceReader returns the ClusterRoleBinding required for the metrics server to read all required resources
func ClusterRoleBindingResourceReader(_ resources.ClusterRoleBindingDataProvider, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	var crb *rbacv1.ClusterRoleBinding
	if existing != nil {
		crb = existing
	} else {
		crb = &rbacv1.ClusterRoleBinding{}
	}

	crb.Name = resources.MetricsServerResourceReaderClusterRoleBindingName
	crb.Labels = resources.BaseAppLabel(name, nil)

	crb.RoleRef = rbacv1.RoleRef{
		Name:     resources.MetricsServerClusterRoleName,
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
	}
	crb.Subjects = []rbacv1.Subject{
		{
			Kind:     "User",
			Name:     resources.MetricsServerCertUsername,
			APIGroup: rbacv1.GroupName,
		},
	}
	return crb, nil
}

// ClusterRoleBindingAuthDelegator returns the ClusterRoleBinding required for the metrics server to create token review requests
func ClusterRoleBindingAuthDelegator(_ resources.ClusterRoleBindingDataProvider, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	var crb *rbacv1.ClusterRoleBinding
	if existing != nil {
		crb = existing
	} else {
		crb = &rbacv1.ClusterRoleBinding{}
	}

	crb.Name = resources.MetricsServerAuthDelegatorClusterRoleBindingName
	crb.Labels = resources.BaseAppLabel(name, nil)

	crb.RoleRef = rbacv1.RoleRef{
		Name:     "system:auth-delegator",
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
	}
	crb.Subjects = []rbacv1.Subject{
		{
			Kind:     "User",
			Name:     resources.MetricsServerCertUsername,
			APIGroup: rbacv1.GroupName,
		},
	}
	return crb, nil
}
