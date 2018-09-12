package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultRoleBinding returns the RoleBinding for the machine-controller.
// It has to be put into the user-cluster.
func DefaultRoleBinding(_ resources.RoleBindingDataProvider, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	// RoleBindingDataProvider actually not needed, no ownerrefs set in user-cluster
	return createRoleBinding(existing, metav1.NamespaceDefault)
}

// KubeSystemRoleBinding returns the RoleBinding for the machine-controller in kube-system ns.
// It has to be put into the user-cluster.
func KubeSystemRoleBinding(_ resources.RoleBindingDataProvider, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	return createRoleBinding(existing, metav1.NamespaceSystem)
}

// KubePublicRoleBinding returns the RoleBinding for the machine-controller in kube-public ns.
// It has to be put into the user-cluster.
func KubePublicRoleBinding(_ resources.RoleBindingDataProvider, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	return createRoleBinding(existing, metav1.NamespacePublic)
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
	rb.Labels = resources.BaseAppLabel(name, nil)

	rb.RoleRef = rbacv1.RoleRef{
		Name:     resources.MachineControllerRoleName,
		Kind:     "Role",
		APIGroup: rbacv1.GroupName,
	}
	rb.Subjects = []rbacv1.Subject{
		{
			Kind:      "User",
			Name:      resources.MachineControllerCertUsername,
			Namespace: rb.Namespace,
			APIGroup:  rbacv1.GroupName,
		},
	}
	return rb, nil
}
