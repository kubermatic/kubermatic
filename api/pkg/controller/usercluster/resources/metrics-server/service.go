package metricsserver

import (
	"fmt"
	"net"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceCreator returns the function to reconcile the metrics server service
func ServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.MetricsServerExternalNameServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Namespace = metav1.NamespaceSystem
			se.Labels = resources.BaseAppLabel(Name, nil)

			se.Spec.Type = corev1.ServiceTypeClusterIP
			// It was previously a ExternalName service. Due to validation errors we must empty it.
			se.Spec.ExternalName = ""
			se.Spec.Ports = []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       443,
					TargetPort: intstr.FromInt(443),
				},
			}

			// We don't specify a selector here as we manually manage the endpoints
			return se, nil
		}
	}
}

// EndpointsCreator returns the function to reconcile the endpoints for the metrics server service
func EndpointsCreator(serviceIP net.IP) reconciling.NamedEndpointsCreatorGetter {
	return func() (string, reconciling.EndpointsCreator) {
		return resources.MetricsServerExternalNameServiceName, func(e *corev1.Endpoints) (*corev1.Endpoints, error) {
			e.Subsets = []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: fmt.Sprint(serviceIP),
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Port:     443,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			}
			return e, nil
		}
	}
}
