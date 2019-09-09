package openshift

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	roleName = "system:openshift:sa-leader-election-configmaps"
)

// KubeSystemRoleCreator returns the func to create/update the Role for the machine controller to allow reading secrets
func KubeSchedulerRoleCreatorGetter() (string, reconciling.RoleCreator) {
	return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
		r.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs: []string{
					"get",
					"create",
					"update",
				},
			},
		}
		return r, nil
	}
}

func KubeSchedulerRoleBindingCreatorGetter() (string, reconciling.RoleBindingCreator) {
	return resources.MachineControllerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
		rb.RoleRef = rbacv1.RoleRef{
			Name:     roleName,
			Kind:     "Role",
			APIGroup: rbacv1.GroupName,
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				Name:     resources.SchedulerCertUsername,
				APIGroup: rbacv1.GroupName,
			},
		}
		return rb, nil
	}
}
