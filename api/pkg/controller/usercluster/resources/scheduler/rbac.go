package scheduler

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

// RoleBindingAuthDelegator creates the RoleBinding which is needed for extension apiserver which do auth delegation
func RoleBindingAuthDelegator() resources.NamedRoleBindingCreatorGetter {
	return resources.RoleBindingAuthenticationReaderCreator("system:kube-scheduler")
}

// ClusterRoleBindingAuthDelegatorCreator creates the ClusterRoleBinding which is needed for extension apiserver which do auth delegation
func ClusterRoleBindingAuthDelegatorCreator() resources.NamedClusterRoleBindingCreatorGetter {
	return resources.ClusterRoleBindingAuthDelegatorCreator("system:kube-scheduler")
}
