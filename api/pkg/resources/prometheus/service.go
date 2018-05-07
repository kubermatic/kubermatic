package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Service returns a service for the prometheus
func Service(data *resources.TemplateData) (*corev1.Service, error) {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          map[string]string{},
			Annotations:     map[string]string{},
			OwnerReferences: []metav1.OwnerReference{data.GetClusterRef()},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector: map[string]string{
				"app":     name,
				"cluster": data.Cluster.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       9090,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("web"),
				},
			},
		},
	}, nil
}
