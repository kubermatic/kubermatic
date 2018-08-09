package prometheus

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

	se.Name = resources.PrometheusServiceName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	se.Labels = resources.BaseAppLabel(name, nil)
	// We need to set cluster: user for the ServiceMonitor which federates metrics8
	se.Labels["cluster"] = "user"

	se.Spec.ClusterIP = "None"
	se.Spec.Selector = map[string]string{
		resources.AppLabelKey: "prometheus",
		"cluster":             data.Cluster.Name,
	}
	se.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "web",
			Port:       9090,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("web"),
		},
	}

	return se, nil
}
