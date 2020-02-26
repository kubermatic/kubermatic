package prometheus

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceCreator returns the function to reconcile the prometheus service used for federation
func ServiceCreator(data *resources.TemplateData) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return name, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = name
			se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
			se.Labels = resources.BaseAppLabels(name, nil)
			// We need to set cluster: user for the ServiceMonitor which federates metrics8
			se.Labels["cluster"] = "user"

			se.Spec.ClusterIP = "None"
			se.Spec.Selector = map[string]string{
				resources.AppLabelKey: "prometheus",
				"cluster":             data.Cluster().Name,
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
	}
}
