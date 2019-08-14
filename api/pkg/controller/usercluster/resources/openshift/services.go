package openshift

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func OpenshiftAPIServicecreatorGetter() (string, reconciling.ServiceCreator) {
	return "api", func(s *corev1.Service) (*corev1.Service, error) {
		s.Spec.Selector = nil
		s.Spec.Type = corev1.ServiceTypeClusterIP
		s.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "openshift-apiserver",
				Protocol:   corev1.ProtocolTCP,
				Port:       443,
				TargetPort: intstr.FromInt(8443),
			},
		}
		return s, nil
	}
}
