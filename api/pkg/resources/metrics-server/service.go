package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Service returns a service for the metrics server
func Service(data resources.ServiceDataProvider, existing *corev1.Service) (*corev1.Service, error) {
	se := existing
	if se == nil {
		se = &corev1.Service{}
	}

	se.Name = resources.MetricsServerServiceName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	labels := resources.BaseAppLabel(name, nil)
	se.Labels = labels

	se.Spec.Selector = labels
	se.Spec.Ports = []corev1.ServicePort{
		{
			Port:       443,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("https"),
		},
	}

	return se, nil
}
