package kubernetesdashboard

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// ServiceCreator creates the service for the dashboard-metrics-scraper
func ServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.MetricsScraperServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Name = resources.MetricsScraperServiceName
			s.Labels = resources.BaseAppLabels(scraperName, nil)
			s.Spec.Selector = resources.BaseAppLabels(scraperName, nil)
			s.Spec.Ports = []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       8000,
					TargetPort: intstr.FromInt(8000),
				},
			}
			return s, nil
		}
	}
}
