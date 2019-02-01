package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceCreator returns the function to reconcile the external OpenVPN service
func ServiceCreator() resources.ServiceCreator {
	return func(se *corev1.Service) (*corev1.Service, error) {
		se.Name = resources.OpenVPNServerServiceName
		se.Labels = resources.BaseAppLabel(name, nil)
		se.Annotations = map[string]string{
			"nodeport-proxy.k8s.io/expose": "true",
		}
		se.Spec.Selector = map[string]string{
			resources.AppLabelKey: name,
		}
		se.Spec.Type = corev1.ServiceTypeNodePort
		if len(se.Spec.Ports) == 0 {
			se.Spec.Ports = make([]corev1.ServicePort, 1)
		}

		se.Spec.Ports[0].Name = "secure"
		se.Spec.Ports[0].Port = 1194
		se.Spec.Ports[0].Protocol = corev1.ProtocolTCP
		se.Spec.Ports[0].TargetPort = intstr.FromInt(1194)

		return se, nil
	}
}
