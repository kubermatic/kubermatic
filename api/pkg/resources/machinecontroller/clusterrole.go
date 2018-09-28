package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRole returns a cluster role for the machine controller (user-cluster)
func ClusterRole(_ resources.ClusterRoleDataProvider, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	r := existing
	if r == nil {
		r = &rbacv1.ClusterRole{}
	}

	r.Name = resources.MachineControllerClusterRoleName
	r.Labels = resources.BaseAppLabel(name, nil)

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"apiextensions.k8s.io"},
			Resources: []string{"customresourcedefinitions"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups:     []string{"apiextensions.k8s.io"},
			Resources:     []string{"customresourcedefinitions"},
			ResourceNames: []string{"machines.machine.k8s.io"},
			Verbs:         []string{"delete"},
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
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"create", "patch"},
		},
		{
			APIGroups: []string{"cluster.k8s.io"},
			Resources: []string{"machines", "machinesets", "machinesets/status", "machinedeployments", "machinedeployments/status", "clusters", "clusters/status"},
			Verbs:     []string{"*"},
		},
	}
	return r, nil
}
