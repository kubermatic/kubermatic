package vpnsidecar

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// DnatControllerClusterRoleBinding returns a ClusterRoleBinding for the kubeletdnat-controller.
// It has to be put into the user-cluster.
func DnatControllerClusterRoleBinding(data *resources.TemplateData, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	return createClusterRoleBinding(existing, "controller",
		resources.KubeletDnatControllerClusterRoleName, rbacv1.Subject{
			Kind:     "User",
			Name:     resources.KubeletDnatControllerCertUsername,
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

	crb.Name = fmt.Sprintf("%s:%s", resources.KubeletDnatControllerClusterRoleBindingName, crbSuffix)

	crb.RoleRef = rbacv1.RoleRef{
		Name:     cRoleRef,
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
	}
	crb.Subjects = []rbacv1.Subject{subj}
	return crb, nil
}
