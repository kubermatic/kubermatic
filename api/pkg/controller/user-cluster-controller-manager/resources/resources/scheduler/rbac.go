package scheduler

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// RoleBindingAuthDelegator creates the RoleBinding which is needed for extension apiserver which do auth delegation
func RoleBindingAuthDelegator() reconciling.NamedRoleBindingCreatorGetter {
	return resources.RoleBindingAuthenticationReaderCreator("system:kube-scheduler")
}

// ClusterRoleBindingAuthDelegatorCreator creates the ClusterRoleBinding which is needed for extension apiserver which do auth delegation
func ClusterRoleBindingAuthDelegatorCreator() reconciling.NamedClusterRoleBindingCreatorGetter {
	return resources.ClusterRoleBindingAuthDelegatorCreator("system:kube-scheduler")
}
