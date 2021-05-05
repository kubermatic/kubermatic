/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metricsserver

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
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
