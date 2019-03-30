package metricsserver

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalNameServiceCreator returns the function to reconcile the metrics server service
func ExternalNameServiceCreator(namespace string) resources.NamedServiceCreatorGetter {
	return func() (string, resources.ServiceCreator) {
		return resources.MetricsServerExternalNameServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Namespace = metav1.NamespaceSystem
			se.Labels = resources.BaseAppLabel(Name, nil)

			se.Spec.Type = corev1.ServiceTypeExternalName
			se.Spec.ExternalName = fmt.Sprintf("%s.%s.svc.cluster.local", resources.MetricsServerServiceName, namespace)

			return se, nil
		}
	}
}
