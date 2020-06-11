package controllermanager

import (
	"github.com/kubermatic/kubermatic/pkg/resources"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"
)

// RoleBindingAuthDelegator creates the RoleBinding which is needed for extension apiserver which do auth delegation
func RoleBindingAuthDelegator() reconciling.NamedRoleBindingCreatorGetter {
	return resources.RoleBindingAuthenticationReaderCreator("system:kube-controller-manager")
}

// ClusterRoleBindingAuthDelegator creates the ClusterRoleBinding which is needed for extension apiserver which do auth delegation
func ClusterRoleBindingAuthDelegator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return resources.ClusterRoleBindingAuthDelegatorCreator("system:kube-controller-manager")
}
