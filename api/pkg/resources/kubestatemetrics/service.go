package kubestatemetrics

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Service returns a service for kube-state-metrics
func Service(data resources.ServiceDataProvider, existing *corev1.Service) (*corev1.Service, error) {
	se := existing
	if se == nil {
		se = &corev1.Service{}
	}

	se.Name = resources.KubeStateMetricsServiceName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	se.Labels = resources.BaseAppLabel(name, nil)
	se.Annotations = map[string]string{
		"prometheus.io/scrape": "true",
	}

	se.Spec.ClusterIP = "None"
	se.Spec.Selector = map[string]string{
		resources.AppLabelKey: name,
		"cluster":             data.Cluster().Name,
	}
	se.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "metrics",
			Port:       8080,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("metrics"),
		},
		{
			Name:       "telemetry",
			Port:       8081,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("telemetry"),
		},
	}

	return se, nil
}
