package ipamcontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRole returns a cluster role for the ipam controller
func ClusterRole(data *resources.TemplateData, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	var r *rbacv1.ClusterRole
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.ClusterRole{}
	}

	r.Name = resources.IPAMControllerClusterRoleName

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"machine.k8s.io"},
			Resources: []string{"machines"},
			Verbs:     []string{"*"},
		},
	}
	return r, nil
}
