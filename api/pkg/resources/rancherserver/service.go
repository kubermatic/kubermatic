package rancherserver

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/nodeportproxy"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// ServiceCreator creates the service for rancher server
func ServiceCreator(exposeStrategy corev1.ServiceType) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.RancherServerServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Name = resources.RancherServerServiceName
			s.Labels = resources.BaseAppLabels(resources.RancherStatefulSetName, nil)
			if s.Annotations == nil {
				s.Annotations = map[string]string{}
			}
			if exposeStrategy == corev1.ServiceTypeNodePort {
				s.Annotations["nodeport-proxy.k8s.io/expose"] = "true"
				delete(s.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			} else {
				s.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"
				delete(s.Annotations, "nodeport-proxy.k8s.io/expose")
			}
			s.Spec.Selector = resources.BaseAppLabels(resources.RancherStatefulSetName, nil)
			s.Spec.Type = corev1.ServiceTypeNodePort
			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = make([]corev1.ServicePort, 2)
			}

			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromInt(80)
			s.Spec.Ports[0].Name = "http"

			s.Spec.Ports[1].Protocol = corev1.ProtocolTCP
			s.Spec.Ports[1].Port = 443
			s.Spec.Ports[1].TargetPort = intstr.FromInt(443)
			s.Spec.Ports[1].Name = "https"

			return s, nil
		}
	}
}
