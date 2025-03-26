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

package kubernetesdashboard

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceReconciler creates the service for the dashboard-metrics-scraper.
func ServiceReconciler(ipFamily kubermaticv1.IPFamily) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return resources.MetricsScraperServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Name = resources.MetricsScraperServiceName
			s.Labels = resources.BaseAppLabels(scraperName, nil)
			s.Spec.Selector = resources.BaseAppLabels(scraperName, nil)
			s.Spec.Ports = []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       8000,
					TargetPort: intstr.FromInt(8000),
				},
			}
			if ipFamily == kubermaticv1.IPFamilyDualStack {
				dsPolicy := corev1.IPFamilyPolicyPreferDualStack
				s.Spec.IPFamilyPolicy = &dsPolicy
			}
			return s, nil
		}
	}
}
