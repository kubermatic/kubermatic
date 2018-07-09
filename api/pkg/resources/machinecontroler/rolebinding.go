package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleBinding returns the RoleBinding for the machine-controller.
// It has to be put into the user-cluster.
func RoleBinding(data *resources.TemplateData, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	// TemplateData actually not needed, no ownerrefs set in user-cluster
	return createRoleBinding(existing, "default")
}
func KubeSystemRoleBinding(data *resources.TemplateData, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	return createRoleBinding(existing, "kube-system")
}

func KubePublicRoleBinding(data *resources.TemplateData, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	return createRoleBinding(existing, "kube-public")
}

func createRoleBinding(existing *rbacv1.RoleBinding, namespace string) (*rbacv1.RoleBinding, error) {
	var rb *rbacv1.RoleBinding
	if existing != nil {
		rb = existing
	} else {
		rb = &rbacv1.RoleBinding{}
	}

	rb.Name = resources.MachineControllerRoleBindingName
	rb.Namespace = namespace
	rb.Labels = resources.GetLabels(name)

	rb.RoleRef = rbacv1.RoleRef{
		Name:     resources.MachineControllerRoleName,
		Kind:     "Role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	rb.Subjects = []rbacv1.Subject{
		{
			Kind:      "User",
			Name:      resources.MachineControllerCertUsername,
			Namespace: rb.Namespace,
		},
	}
	return rb, nil
}
