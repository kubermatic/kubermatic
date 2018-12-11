package ipamcontroller

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

const name = "ipam-controller"

// ClusterRole returns a cluster role for the ipam controller
func ClusterRole(_ resources.ClusterRoleDataProvider, existing *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	r := existing
	if r == nil {
		r = &rbacv1.ClusterRole{}
	}

	r.Name = resources.IPAMControllerClusterRoleName
	r.Labels = resources.BaseAppLabel(name, nil)
	r.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"cluster.k8s.io"},
			Resources: []string{"machines"},
			Verbs:     []string{"list", "get", "watch", "update", "initialize"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"endpoints", "events"},
			Verbs:     []string{"*"},
		},
	}
	return r, nil
}

// ClusterRoleBinding returns a ClusterRoleBinding for the ipam-controller.
// It has to be put into the user-cluster.
func ClusterRoleBinding(_ resources.ClusterRoleBindingDataProvider, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	// TemplateData actually not needed, no ownerrefs set in user-cluster
	return createClusterRoleBinding(existing, "controller",
		resources.IPAMControllerClusterRoleName, rbacv1.Subject{
			Kind:     "User",
			Name:     resources.IPAMControllerCertUsername,
			APIGroup: rbacv1.GroupName,
		})
}

func createClusterRoleBinding(existing *rbacv1.ClusterRoleBinding, crbSuffix, cRoleRef string, subj rbacv1.Subject) (*rbacv1.ClusterRoleBinding, error) {
	crb := existing
	if crb == nil {
		crb = &rbacv1.ClusterRoleBinding{}
	}

	crb.Name = fmt.Sprintf("%s:%s", resources.IPAMControllerClusterRoleBindingName, crbSuffix)
	crb.Labels = resources.BaseAppLabel(name, nil)

	crb.RoleRef = rbacv1.RoleRef{
		Name:     cRoleRef,
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
	}
	crb.Subjects = []rbacv1.Subject{subj}
	return crb, nil
}
