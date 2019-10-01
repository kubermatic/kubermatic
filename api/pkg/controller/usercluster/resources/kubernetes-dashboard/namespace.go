package kubernetesdashboard

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// NamespaceCreator TODO(floreks)
func NamespaceCreator() (string, reconciling.NamespaceCreator) {
	return Namespace, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
		return ns, nil
	}
}
