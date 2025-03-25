/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceReconciler returns the function to reconcile the user cluster metrics-server service.
func ServiceReconciler(ipFamily kubermaticv1.IPFamily) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return resources.MetricsServerServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = resources.MetricsServerServiceName
			labels := resources.BaseAppLabels(resources.MetricsServerDeploymentName, nil)
			se.Labels = labels
			se.Spec.Selector = labels

			se.Spec.Type = corev1.ServiceTypeClusterIP
			se.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "https",
					Port:       443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("https"),
				},
			}

			if ipFamily == kubermaticv1.IPFamilyDualStack {
				dsPolicy := corev1.IPFamilyPolicyPreferDualStack
				se.Spec.IPFamilyPolicy = &dsPolicy
			}

			return se, nil
		}
	}
}
