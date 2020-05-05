package coredns

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceCreator creates the service for the CoreDNS
func ServiceCreator(DNSClusterIP string) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		labels := map[string]string{
			"kubernetes.io/cluster-service": "true",
			"kubernetes.io/name":            "KubeDNS",
		}
		return resources.CoreDNSServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Name = resources.CoreDNSServiceName
			s.Labels = resources.BaseAppLabels(resources.CoreDNSDeploymentName, labels)
			s.Spec.Selector = resources.BaseAppLabels(resources.CoreDNSDeploymentName, nil)
			s.Spec.ClusterIP = DNSClusterIP
			s.Spec.Ports = []corev1.ServicePort{
				{
					Name:     "dns-tcp",
					Protocol: corev1.ProtocolTCP,
					Port:     53,
				},
				{
					Name:     "dns",
					Protocol: corev1.ProtocolUDP,
					Port:     53,
				},
			}
			return s, nil
		}
	}
}
