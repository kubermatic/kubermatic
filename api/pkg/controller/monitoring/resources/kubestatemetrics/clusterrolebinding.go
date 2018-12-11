package kubestatemetrics

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBinding returns the ClusterRoleBinding required for Kube-State-Metrics
func ClusterRoleBinding(_ resources.ClusterRoleBindingDataProvider, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	var crb *rbacv1.ClusterRoleBinding
	if existing != nil {
		crb = existing
	} else {
		crb = &rbacv1.ClusterRoleBinding{}
	}

	crb.Name = resources.KubeStateMetricsClusterRoleBindingName
	crb.Labels = resources.BaseAppLabel(name, nil)

	crb.RoleRef = rbacv1.RoleRef{
		Name:     resources.KubeStateMetricsClusterRoleName,
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
	}
	crb.Subjects = []rbacv1.Subject{
		{
			Kind:     "User",
			Name:     resources.KubeStateMetricsCertUsername,
			APIGroup: rbacv1.GroupName,
		},
	}
	return crb, nil
}
