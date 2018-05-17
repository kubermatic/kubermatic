package etcd

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Service returns a service for the etcd
func Service(data *resources.TemplateData, existing *corev1.Service) (*corev1.Service, error) {
	var se *corev1.Service
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Service{}
	}

	se.Name = resources.EtcdClientServiceName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	se.Spec.Type = corev1.ServiceTypeClusterIP
	se.Spec.Selector = map[string]string{
		resources.AppLabelKey: name,
	}
	se.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "client",
			Port:       2379,
			TargetPort: intstr.FromInt(2379),
			Protocol:   corev1.ProtocolTCP,
		},
	}

	return se, nil
}
