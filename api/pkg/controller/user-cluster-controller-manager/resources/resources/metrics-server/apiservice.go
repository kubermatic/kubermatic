package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Name = "metrics-server"
)

// APIServiceCreator returns the func to create/update the APIService used by the metrics-server
func APIServiceCreator(caBundle []byte) reconciling.NamedAPIServiceCreatorGetter {
	return func() (string, reconciling.APIServiceCreator) {
		return resources.MetricsServerAPIServiceName, func(se *apiregistrationv1beta1.APIService) (*apiregistrationv1beta1.APIService, error) {
			labels := resources.BaseAppLabels(Name, nil)
			se.Labels = labels

			if se.Spec.Service == nil {
				se.Spec.Service = &apiregistrationv1beta1.ServiceReference{}
			}
			se.Spec.Service.Namespace = metav1.NamespaceSystem
			se.Spec.Service.Name = resources.MetricsServerExternalNameServiceName
			se.Spec.Group = "metrics.k8s.io"
			se.Spec.Version = "v1beta1"
			se.Spec.InsecureSkipTLSVerify = false
			se.Spec.CABundle = caBundle
			se.Spec.GroupPriorityMinimum = 100
			se.Spec.VersionPriority = 100

			return se, nil
		}
	}
}
