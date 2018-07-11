package controllermanager

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleBinding returns the RoleBinding for the controller-manager.
// It has to be put into the user-cluster.
func SystemBootstrapSignerRoleBinding(data *resources.TemplateData, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	return createRoleBinding(existing, "kube-system", "system:controller:bootstrap-signer")
}
func PublicBootstrapSignerRoleBinding(data *resources.TemplateData, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	return createRoleBinding(existing, "kube-public", "system:controller:bootstrap-signer")
}

// createRoleBinding creates a binding of a role in the given namespace
// onto the ControllerManager's user (kubeconfig cert)
func createRoleBinding(existing *rbacv1.RoleBinding, namespace, roleRef string) (*rbacv1.RoleBinding, error) {
	var rb *rbacv1.RoleBinding
	if existing != nil {
		rb = existing
	} else {
		rb = &rbacv1.RoleBinding{}
	}

	rb.Name = resources.ControllerManagerRoleBindingName
	rb.Namespace = namespace
	rb.Labels = resources.GetLabels(name)

	rb.RoleRef = rbacv1.RoleRef{
		Name:     roleRef,
		Kind:     "Role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	rb.Subjects = []rbacv1.Subject{
		{
			Kind:      "User",
			Name:      resources.ControllerManagerCertUsername,
			Namespace: rb.Namespace,
		},
	}
	return rb, nil
}
