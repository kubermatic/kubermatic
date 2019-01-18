package controllermanager

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleBindingAuthDelegator creates the RoleBinding which is needed for extension apiserver which do auth delegation
func RoleBindingAuthDelegator(data resources.RoleBindingDataProvider, existing *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	return resources.RoleBindingAuthenticationReaderCreator("system:kube-controller-manager")(data, existing)
}

// ClusterRoleBindingAuthDelegator creates the ClusterRoleBinding which is needed for extension apiserver which do auth delegation
func ClusterRoleBindingAuthDelegator(data resources.ClusterRoleBindingDataProvider, existing *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	return resources.ClusterRoleBindingAuthDelegatorCreator("system:kube-controller-manager")(data, existing)
}
