package controllermanager

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// AdminClusterRoleBinding allows just everything.
func AdminClusterRoleBinding(data *resources.TemplateData, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	var crb *rbacv1.ClusterRoleBinding
	if existing != nil {
		crb = existing
	} else {
		crb = &rbacv1.ClusterRoleBinding{}
	}

	crb.Name = resources.ControllerManagerClusterRoleBindingName
	crb.Labels = resources.GetLabels(name)
	crb.RoleRef = rbacv1.RoleRef{
		Name:     "cluster-admin",
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
	}
	crb.Subjects = []rbacv1.Subject{
		{
			Kind:     "User",
			Name:     resources.ControllerManagerCertUsername,
			APIGroup: rbacv1.GroupName,
		},
	}
	return crb, nil
}
