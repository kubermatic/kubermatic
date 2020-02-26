package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceCreator returns the function to reconcile the seed cluster internal metrics-server service
func ServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.MetricsServerServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = resources.MetricsServerServiceName
			labels := resources.BaseAppLabels(name, nil)
			se.Labels = labels

			se.Spec.Selector = labels
			se.Spec.Ports = []corev1.ServicePort{
				{
					Port:       443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(443),
				},
			}

			return se, nil
		}
	}
}
