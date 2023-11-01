package clusterbackup

import (
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	NamespaceName      = "velero"
	serviceAccountName = "velero"
)

// NamespaceReconciler creates the namespace for the Kubernetes Dashboard.
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
