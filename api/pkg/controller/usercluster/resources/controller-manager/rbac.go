package controllermanager

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

// RoleBindingAuthDelegator creates the RoleBinding which is needed for extension apiserver which do auth delegation
func RoleBindingAuthDelegator() resources.NamedRoleBindingCreatorGetter {
	return resources.RoleBindingAuthenticationReaderCreator("system:kube-controller-manager")
}

// ClusterRoleBindingAuthDelegator creates the ClusterRoleBinding which is needed for extension apiserver which do auth delegation
func ClusterRoleBindingAuthDelegator() resources.NamedClusterRoleBindingCreatorGetter {
	return resources.ClusterRoleBindingAuthDelegatorCreator("system:kube-controller-manager")
}
