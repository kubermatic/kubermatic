package rancherserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleBindingCreator returns the ClusterRoleBinding required by rancher server
func ClusterRoleBindingCreator(clusterNamespace string) reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return resources.RancherServerClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabel(resources.RancherStatefulSetName, nil)

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Name:     "cluster-admin", // nope! should use something with lower privileges
				Kind:     "ClusterRole",
			}
			crb.Subjects = []rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      resources.RancherServerServiceAccountName,
				Namespace: clusterNamespace,
			}}
			return crb, nil
		}
	}

}
