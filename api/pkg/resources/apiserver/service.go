package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Service returns the internal service for the apiserver
func Service(data *resources.TemplateData, existing *corev1.Service) (*corev1.Service, error) {
	var se *corev1.Service
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Service{}
	}

	se.Name = resources.ApiserverInternalServiceName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	se.Labels = resources.BaseAppLabel(name, nil)

	se.Spec.Type = corev1.ServiceTypeClusterIP
	se.Spec.Selector = map[string]string{
		resources.AppLabelKey: name,
	}
	se.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "insecure",
			Port:       8080,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(8080),
		},
	}

	return se, nil
}

// ExternalService returns the internal service for the apiserver
func ExternalService(data *resources.TemplateData, existing *corev1.Service) (*corev1.Service, error) {
	var se *corev1.Service
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Service{}
	}

	se.Name = resources.ApiserverExternalServiceName
	se.Annotations = map[string]string{
		"nodeport-proxy.k8s.io/expose": "true",
	}
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	se.Spec.Type = corev1.ServiceTypeNodePort
	se.Spec.Selector = map[string]string{
		resources.AppLabelKey: name,
	}

	if len(se.Spec.Ports) == 0 {
		se.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "secure",
				Port:       443,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(443),
			},
		}

		return se, nil
	}

	se.Spec.Ports[0].Name = "secure"
	se.Spec.Ports[0].Port = se.Spec.Ports[0].NodePort
	se.Spec.Ports[0].TargetPort = intstr.FromInt(int(se.Spec.Ports[0].NodePort))
	se.Spec.Ports[0].Protocol = corev1.ProtocolTCP

	return se, nil
}
