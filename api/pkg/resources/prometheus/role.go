package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Role returns a role for the prometheus
func Role(data resources.RoleDataProvider, existing *rbacv1.Role) (*rbacv1.Role, error) {
	var r *rbacv1.Role
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.Role{}
	}

	r.Name = resources.PrometheusRoleName
	r.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	r.Labels = resources.BaseAppLabel(name, nil)

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{
				"nodes",
				"services",
				"endpoints",
				"pods",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
			},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"get"},
		},
	}
	return r, nil
}
