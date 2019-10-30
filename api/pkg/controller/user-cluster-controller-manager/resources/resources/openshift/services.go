package openshift

import (
	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/openshift/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

func APIServicecreatorGetterFactory(clusterNS string) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return "api", func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Selector = nil
			s.Spec.Type = corev1.ServiceTypeExternalName
			s.Spec.Ports = nil
			s.Spec.ExternalName = openshiftresources.OpenshiftAPIServerServiceName + "." + clusterNS + ".svc.cluster.local"
			return s, nil
		}
	}
}
