package resources

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceReconciler returns the function to create and update the Kyverno cleanup controller service.
func ServiceReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return name, func(s *corev1.Service) (*corev1.Service, error) {
			s.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.Selector = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
			}

			s.Spec.Ports = []corev1.ServicePort{
				{
					Name:        "https",
					Port:        443,
					Protocol:    corev1.ProtocolTCP,
					TargetPort:  intstr.FromString("https"),
					AppProtocol: &[]string{"https"}[0],
				},
			}

			return s, nil
		}
	}
}

// MetricsServiceReconciler returns the function to create and update the Kyverno cleanup controller metrics service.
func MetricsServiceReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return name + "-metrics", func(s *corev1.Service) (*corev1.Service, error) {
			s.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}

			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.Selector = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
			}

			s.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "metrics-port",
					Port:       8000,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8000),
				},
			}

			return s, nil
		}
	}
}
