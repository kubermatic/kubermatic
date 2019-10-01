package metricsscraper

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// ServiceAccountCreator TODO(floreks)
func ServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return resources.MetricsScraperServiceAccountUsername, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = resources.BaseAppLabel(name, nil)
			sa.Name = resources.MetricsScraperServiceAccountUsername
			return sa, nil
		}
	}
}
