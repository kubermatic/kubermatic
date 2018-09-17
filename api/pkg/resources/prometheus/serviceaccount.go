package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccount returns the ServiceAccount used by Prometheus.
func ServiceAccount(data resources.ServiceAccountDataProvider, existing *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
	var sa *corev1.ServiceAccount
	if existing != nil {
		sa = existing
	} else {
		sa = &corev1.ServiceAccount{}
	}

	sa.Name = resources.PrometheusServiceAccountName
	sa.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	return sa, nil
}
