package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
)

// ServiceCreator returns the function to reconcile the DNS service
func ServiceCreator() resources.ServiceCreator {
	return func(se *corev1.Service) (*corev1.Service, error) {
		se.Name = resources.MetricsServerServiceName
		labels := resources.BaseAppLabel(name, nil)
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
