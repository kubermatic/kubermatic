package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRole returns a cluster role for the machine controller (user-cluster)
func ClusterRole(data *resources.TemplateData, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	var r *rbacv1.ClusterRole
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.ClusterRole{}
	}

	r.Name = resources.MachineControllerClusterRoleName
	r.Labels = resources.BaseAppLabel(name)

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"apiextensions.k8s.io"},
			Resources: []string{"customresourcedefinitions"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups:     []string{"apiextensions.k8s.io"},
			Resources:     []string{"customresourcedefinitions"},
			ResourceNames: []string{"machines.machine.k8s.io"},
			Verbs:         []string{"*"},
		},
		{
			APIGroups: []string{"machine.k8s.io"},
			Resources: []string{"machines"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"*"},
		},
	}
	return r, nil
}
