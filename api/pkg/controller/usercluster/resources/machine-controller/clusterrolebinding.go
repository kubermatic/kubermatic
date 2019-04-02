package machinecontroller

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBinding returns a ClusterRoleBinding for the machine-controller.
func ClusterRoleBindingCreator() resources.NamedClusterRoleBindingCreatorGetter {
	// TemplateData actually not needed, no ownerrefs set in user-cluster
	return createClusterRoleBindingCreator("controller",
		resources.MachineControllerClusterRoleName, rbacv1.Subject{
			Kind:     "User",
			Name:     resources.MachineControllerCertUsername,
			APIGroup: rbacv1.GroupName,
		})
}

// NodeBootstrapperClusterRoleBinding returns a ClusterRoleBinding for the machine-controller.
func NodeBootstrapperClusterRoleBindingCreator() resources.NamedClusterRoleBindingCreatorGetter {
	return createClusterRoleBindingCreator("kubelet-bootstrap",
		"system:node-bootstrapper", rbacv1.Subject{
			Kind:     "Group",
			Name:     "system:bootstrappers:machine-controller:default-node-token",
			APIGroup: rbacv1.GroupName,
		})
}

// NodeSignerClusterRoleBindingCreator returns a ClusterRoleBinding for the machine-controller.
func NodeSignerClusterRoleBindingCreator() resources.NamedClusterRoleBindingCreatorGetter {
	return createClusterRoleBindingCreator("node-signer",
		"system:certificates.k8s.io:certificatesigningrequests:nodeclient", rbacv1.Subject{
			Kind:     "Group",
			Name:     "system:bootstrappers:machine-controller:default-node-token",
			APIGroup: rbacv1.GroupName,
		})
}

func createClusterRoleBindingCreator(crbSuffix, cRoleRef string, subj rbacv1.Subject) resources.NamedClusterRoleBindingCreatorGetter {
	return func() (string, resources.ClusterRoleBindingCreator) {
		return fmt.Sprintf("%s:%s", resources.MachineControllerClusterRoleBindingName, crbSuffix), func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabel(Name, nil)

			crb.RoleRef = rbacv1.RoleRef{
				Name:     cRoleRef,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{subj}
			return crb, nil
		}
	}
}
