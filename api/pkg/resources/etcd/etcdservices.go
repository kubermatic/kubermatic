package etcd

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DiscoveryService returns a service for the etcd peers to find each other?
func DiscoveryService(data *resources.TemplateData, existing *corev1.Service) (*corev1.Service, error) {
	var se *corev1.Service
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Service{}
	}

	se.Name = resources.EtcdServiceName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	se.Annotations = map[string]string{
		"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
	}
	se.Spec.ClusterIP = "None"
	se.Spec.Selector = resources.BaseAppLabel(name)

	se.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "client",
			Port:       2379,
			TargetPort: intstr.FromInt(2379),
			Protocol:   corev1.ProtocolTCP,
		},
		{
			Name:       "peer",
			Port:       2380,
			TargetPort: intstr.FromInt(2380),
			Protocol:   corev1.ProtocolTCP,
		},
	}

	return se, nil
}

// ClientService returns a service for access by clients (ClusterIP)
func ClientService(data *resources.TemplateData, existing *corev1.Service) (*corev1.Service, error) {
	var se *corev1.Service
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Service{}
	}

	se.Name = resources.EtcdClientServiceName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	se.Spec.Selector = resources.BaseAppLabel(name)

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
