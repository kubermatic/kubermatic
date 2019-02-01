package metricsserver

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceCreator returns the function to reconcile the Prometheus service
func ExternalNameServiceCreator(data resources.ServiceDataProvider) resources.ServiceCreator {
	return func(se *corev1.Service) (*corev1.Service, error) {
		se.Name = resources.MetricsServerExternalNameServiceName
		se.Namespace = metav1.NamespaceSystem
		labels := resources.BaseAppLabel(name, nil)
		se.Labels = labels

		se.Spec.Type = corev1.ServiceTypeExternalName
		se.Spec.ExternalName = fmt.Sprintf("%s.%s.svc.cluster.local", resources.MetricsServerServiceName, data.Cluster().Status.NamespaceName)

		return se, nil
	}
}
