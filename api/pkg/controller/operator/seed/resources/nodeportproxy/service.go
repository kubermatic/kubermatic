package nodeportproxy

import (
	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NB: Changing anything in this service can lead to new LoadBalancers being
// created and IPs changing. This must not happen when customers upgrade Kubermatic,
// as all existing kubeconfigs for user clusters would be broken.

const (
	ServiceName = "nodeport-proxy"
)

func ServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return ServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			// We don't actually manage this service, that is done by the nodeport proxy, we just
			// must make sure that it exists

			s.Spec.Type = corev1.ServiceTypeLoadBalancer
			s.Spec.Selector = map[string]string{
				common.NameLabel: ServiceName,
			}

			// Services need at least one port to be valid, so create it initially.
			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = []corev1.ServicePort{
					{
						Name:       "healthz",
						Port:       EnvoyPort,
						TargetPort: intstr.FromInt(EnvoyPort),
						Protocol:   corev1.ProtocolTCP,
					},
				}
			}

			return s, nil
		}
	}
}
