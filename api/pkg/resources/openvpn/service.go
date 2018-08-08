package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Service returns a service for the prometheus
func Service(data *resources.TemplateData, existing *corev1.Service) (*corev1.Service, error) {
	var se *corev1.Service
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Service{}
	}

	se.Name = resources.OpenVPNServerServiceName
	se.Labels = resources.BaseAppLabel("openvpn")
	se.Annotations = map[string]string{
		"nodeport-proxy.k8s.io/expose": "true",
	}
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	se.Spec.Selector = resources.BaseAppLabel(name)

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
