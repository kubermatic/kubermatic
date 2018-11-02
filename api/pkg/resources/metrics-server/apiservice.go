package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

// APIService returns a APIService for the metrics server - used inside the user cluster
func APIService(existing *apiregistrationv1.APIService) (*apiregistrationv1.APIService, error) {
	se := existing
	if se == nil {
		se = &apiregistrationv1.APIService{}
	}

	se.Name = resources.MetricsServerAPIServiceName
	labels := resources.BaseAppLabel(name, nil)
	se.Labels = labels

	se.Spec.Service = &apiregistrationv1.ServiceReference{
		Namespace: metav1.NamespaceSystem,
		Name:      resources.MetricsServerExternalNameServiceName,
	}
	se.Spec.Group = "metrics.k8s.io"
	se.Spec.Version = "v1beta1"
	se.Spec.InsecureSkipTLSVerify = true
	se.Spec.GroupPriorityMinimum = 100
	se.Spec.VersionPriority = 100

	return se, nil
}
