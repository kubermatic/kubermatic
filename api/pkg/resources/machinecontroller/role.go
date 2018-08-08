package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeSystemRole returns a role for the machine controller. This
// role has to be put in the user-cluster and carries a namespace.
func KubeSystemRole(data *resources.TemplateData, existing *rbacv1.Role) (*rbacv1.Role, error) {
	var r *rbacv1.Role
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.Role{}
	}

	r.Name = resources.MachineControllerRoleName
	r.Namespace = metav1.NamespaceSystem
	r.Labels = resources.BaseAppLabel(name)

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs: []string{
				"create",
				"update",
				"list",
				"watch",
			},
		},
		{
			APIGroups:     []string{""},
			Resources:     []string{"endpoints"},
			ResourceNames: []string{"machine-controller"},
			Verbs:         []string{"*"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"endpoints"},
			Verbs:     []string{"create"},
		},
	}
	return r, nil
}

// KubePublicRole returns a role for the machine controller. This
// role has to be put in the user-cluster and carries a namespace.
func KubePublicRole(data *resources.TemplateData, existing *rbacv1.Role) (*rbacv1.Role, error) {
	var r *rbacv1.Role
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.Role{}
	}

	r.Name = resources.MachineControllerRoleName
	r.Namespace = metav1.NamespacePublic
	r.Labels = resources.BaseAppLabel(name)

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs: []string{
				"get",
				"list",
				"watch",
			},
		},
	}
	return r, nil
}

// Role returns a role for the machine controller. This
// role has to be put in the user-cluster and carries a namespace.
func Role(data *resources.TemplateData, existing *rbacv1.Role) (*rbacv1.Role, error) {
	var r *rbacv1.Role
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.Role{}
	}

	r.Name = resources.MachineControllerRoleName
	r.Namespace = metav1.NamespaceDefault
	r.Labels = resources.BaseAppLabel(name)

	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"endpoints"},
			Verbs: []string{
				"get",
				"list",
				"watch",
			},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs: []string{
				"create",
				"patch",
			},
		},
	}
	return r, nil
}
