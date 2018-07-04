package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterRole returns a cluster role for the machine controller
func ClusterRole(data *resources.TemplateData, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	var r *rbacv1.ClusterRole
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.ClusterRole{}
	}

	r.Name = resources.MachineControllerClusterRoleName
	r.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	r.Labels = resources.GetLabels(name)

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{
				"machines",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
			},
		},
	}
	return r, nil
}
