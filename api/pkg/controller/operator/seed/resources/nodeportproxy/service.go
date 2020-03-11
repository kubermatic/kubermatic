package nodeportproxy

import (
	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
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

func ServiceCreator(cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return ServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeLoadBalancer
			s.Spec.Selector = map[string]string{
				common.NameLabel: ServiceName,
			}

			s.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "healthz",
					Port:       EnvoyPort,
					TargetPort: intstr.FromInt(EnvoyPort),
					Protocol:   corev1.ProtocolTCP,
				},
			}

			return s, nil
		}
	}
}
