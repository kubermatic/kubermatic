package clusterbackup

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// NamespaceReconciler creates the namespace for the Kubernetes Dashboard.
func NamespaceReconciler() (string, reconciling.NamespaceReconciler) {
	return resources.ClusterBackupNamespaceName, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
		return ns, nil
	}
}
