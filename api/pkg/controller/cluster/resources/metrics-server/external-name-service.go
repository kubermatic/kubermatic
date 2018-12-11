package metricsserver

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalNameService returns a ExternalName service for the metrics server - used inside the user cluster
func ExternalNameService(data resources.ServiceDataProvider, existing *corev1.Service) (*corev1.Service, error) {
	se := existing
	if se == nil {
		se = &corev1.Service{}
	}

	se.Name = resources.MetricsServerExternalNameServiceName
	se.Namespace = metav1.NamespaceSystem
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	labels := resources.BaseAppLabel(name, nil)
	se.Labels = labels

	se.Spec.Type = corev1.ServiceTypeExternalName

	se.Spec.ExternalName = fmt.Sprintf("%s.%s.svc.cluster.local", resources.MetricsServerServiceName, data.Cluster().Status.NamespaceName)

	return se, nil
}
