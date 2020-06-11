package machinecontroller

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultRoleBindingCreator returns the func to create/update the RoleBinding for the machine-controller.
func DefaultRoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	// RoleBindingDataProvider actually not needed, no ownerrefs set in user-cluster
	return RoleBindingCreator()
}

// KubeSystemRoleBinding returns the RoleBinding for the machine-controller in kube-system ns.
func KubeSystemRoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return RoleBindingCreator()
}

// KubePublicRoleBinding returns the RoleBinding for the machine-controller in kube-public ns.
func KubePublicRoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return RoleBindingCreator()
}

func RoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.MachineControllerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(machinecontroller.Name, nil)

			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.MachineControllerRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					Name:     resources.MachineControllerCertUsername,
					APIGroup: rbacv1.GroupName,
				},
			}
			return rb, nil
		}
	}
}

// ClusterInfoAnonymousRoleBindingCreator returns a func to create/update the RoleBinding to allow anonymous access to the cluster-info ConfigMap
func ClusterInfoAnonymousRoleBindingCreator() reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return resources.ClusterInfoAnonymousRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Namespace = metav1.NamespacePublic

			rb.RoleRef = rbacv1.RoleRef{
				Name:     resources.ClusterInfoReaderRoleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					APIGroup: rbacv1.GroupName,
					Kind:     rbacv1.UserKind,
					Name:     "system:anonymous",
				},
			}
			return rb, nil
		}
	}
}
