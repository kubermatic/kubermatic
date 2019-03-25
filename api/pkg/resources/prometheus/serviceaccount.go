package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountCreator returns a func to create/update the ServiceAccount used by Prometheus.
func ServiceAccountCreator() resources.NamedServiceAccountCreatorGetter {
	return func() (string, resources.ServiceAccountCreator) {
		return resources.PrometheusServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}
