package gatekeeper

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

// ServiceCreator returns the function to reconcile the gatekeeper service
func ServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.GatekeeperWebhookServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = resources.GatekeeperWebhookServiceName
			labels := resources.BaseAppLabels(controllerName, map[string]string{"gatekeeper.sh/system": "yes"})
			se.Labels = labels

			se.Spec.Selector = gatekeeperControllerLabels
			se.Spec.Ports = []corev1.ServicePort{
				{
					Port:       443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8443),
				},
			}

			return se, nil
		}
	}
}
