package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeSystemRole returns a role for the machine controller. This
// role has to be put in the user-cluster and can thus carry
// a namespace.
func KubeSystemRole(data *resources.TemplateData, existing *rbacv1.Role) (*rbacv1.Role, error) {
	var r *rbacv1.Role
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.Role{}
	}

	r.Name = resources.MachineControllerRoleName
	r.Namespace = "kube-system"
	r.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	r.Labels = resources.GetLabels(name)

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
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			ResourceNames: []string{
				"machine-controller-aws",
				"machine-controller-digitalocean",
				"machine-controller-hetzner",
				"machine-controller-openstack",
				"machine-controller-ssh-key",
				"machine-controller-vsphere",
			},
			Verbs: []string{"get"},
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
func KubePublicRole(data *resources.TemplateData, existing *rbacv1.Role) (*rbacv1.Role, error) {
	var r *rbacv1.Role
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.Role{}
	}

	r.Name = resources.MachineControllerRoleName
	r.Namespace = "kube-public"
	r.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	r.Labels = resources.GetLabels(name)

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
func Role(data *resources.TemplateData, existing *rbacv1.Role) (*rbacv1.Role, error) {
	var r *rbacv1.Role
	if existing != nil {
		r = existing
	} else {
		r = &rbacv1.Role{}
	}

	r.Name = resources.MachineControllerRoleName
	r.Namespace = "default"
	r.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	r.Labels = resources.GetLabels(name)

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
