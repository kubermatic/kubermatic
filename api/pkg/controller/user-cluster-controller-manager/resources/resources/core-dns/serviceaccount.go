package coredns

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccountCreator creates the service account for CoreDNS
func ServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return resources.CoreDNSServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Name = resources.CoreDNSServiceAccountName
			sa.Namespace = metav1.NamespaceSystem
			return sa, nil
		}
	}
}
