package machinecontroller

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBinding returns a ClusterRoleBinding for the machine-controller.
// It has to be put into the user-cluster.
func ClusterRoleBinding(data *resources.TemplateData, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	// TemplateData actually not needed, no ownerrefs set in user-cluster
	return createClusterRoleBinding(existing, "controller",
		resources.MachineControllerClusterRoleName, rbacv1.Subject{
			Kind:     "User",
			Name:     resources.MachineControllerCertUsername,
			APIGroup: rbacv1.GroupName,
		})
}

// NodeBootstrapperClusterRoleBinding returns a ClusterRoleBinding for the machine-controller.
// It has to be put into the user-cluster.
func NodeBootstrapperClusterRoleBinding(data *resources.TemplateData, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	return createClusterRoleBinding(existing, "kubelet-bootstrap",
		"system:node-bootstrapper", rbacv1.Subject{
			Kind:     "Group",
			Name:     "system:bootstrappers:machine-controller:default-node-token",
			APIGroup: rbacv1.GroupName,
		})
}

// NodeSignerClusterRoleBinding returns a ClusterRoleBinding for the machine-controller.
// It has to be put into the user-cluster.
func NodeSignerClusterRoleBinding(data *resources.TemplateData, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	return createClusterRoleBinding(existing, "node-signer",
		"system:certificates.k8s.io:certificatesigningrequests:nodeclient", rbacv1.Subject{
			Kind:     "Group",
			Name:     "system:bootstrappers:machine-controller:default-node-token",
			APIGroup: rbacv1.GroupName,
		})
}

func createClusterRoleBinding(existing *rbacv1.ClusterRoleBinding, crbSuffix, cRoleRef string, subj rbacv1.Subject) (*rbacv1.ClusterRoleBinding, error) {
	var crb *rbacv1.ClusterRoleBinding
	if existing != nil {
		crb = existing
	} else {
		crb = &rbacv1.ClusterRoleBinding{}
	}

	crb.Name = fmt.Sprintf("%s:%s", resources.MachineControllerClusterRoleBindingName, crbSuffix)
	crb.Labels = resources.BaseAppLabel(name)

	crb.RoleRef = rbacv1.RoleRef{
		Name:     cRoleRef,
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
	}
	crb.Subjects = []rbacv1.Subject{subj}
	return crb, nil
}
