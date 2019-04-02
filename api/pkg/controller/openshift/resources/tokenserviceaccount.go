package resources

import (
	"context"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	tokenOwnerServiceAccountName        = "cluster-admin"
	tokenOwnerServiceAccountBindingName = "cluster-admin-serviceaccount"
)

// TokenOwnerServiceAccount is the ServiceAccount that owns the secret which we put onto the
// kubeconfig that is in the seed
func TokenOwnerServiceAccount(_ context.Context) (string, reconciling.ServiceAccountCreator) {
	return tokenOwnerServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

// TokenOwnerServiceAccountClusterRoleBinding is the clusterrolebinding that gives the TokenOwnerServiceAccount
// admin powers
func TokenOwnerServiceAccountClusterRoleBinding(_ context.Context) (string, reconciling.ClusterRoleBindingCreator) {
	return tokenOwnerServiceAccountBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
		crb.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      tokenOwnerServiceAccountName,
				Namespace: metav1.NamespaceSystem,
			},
		}
		crb.RoleRef = rbacv1.RoleRef{
			Name:     "cluster-admin",
			Kind:     "ClusterRole",
			APIGroup: rbacv1.GroupName,
		}
		return crb, nil
	}
}
