package clusterbackup

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	NamespaceName          = "velero"
	serviceAccountName     = "velero"
	clusterRoleBindingName = "velero"
	clusterBackupAppName   = "velero"
)

// NamespaceReconciler creates the namespace for the velero.
func NamespaceReconciler() (string, reconciling.NamespaceReconciler) {
	return NamespaceName, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
		return ns, nil
	}
}

func ServiceAccountReconciler() reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Namespace = NamespaceName
			return sa, nil
		}
	}
}

func ClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return clusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = resources.BaseAppLabels(clusterBackupAppName, nil)

			crb.RoleRef = rbacv1.RoleRef{
				Name:     "cluster-admin",
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: NamespaceName,
				},
			}
			return crb, nil
		}
	}
}
