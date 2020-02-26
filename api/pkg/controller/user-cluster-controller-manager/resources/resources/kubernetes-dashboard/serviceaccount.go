package kubernetesdashboard

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// ServiceAccountCreator creates the service account for the dashboard-metrics-scraper
func ServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return resources.MetricsScraperServiceAccountUsername, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = resources.BaseAppLabels(scraperName, nil)
			sa.Name = resources.MetricsScraperServiceAccountUsername
			return sa, nil
		}
	}
}
