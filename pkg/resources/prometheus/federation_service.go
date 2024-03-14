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

package prometheus

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceReconciler returns the function to reconcile the prometheus service used for federation.
func ServiceReconciler(data *resources.TemplateData) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return name, func(se *corev1.Service) (*corev1.Service, error) {
			se.Labels = resources.BaseAppLabels(name, map[string]string{
				// We need to set cluster: user for the ServiceMonitor which federates metrics
				"cluster": "user",
			})

			se.Spec.ClusterIP = "None"
			se.Spec.Selector = resources.BaseAppLabels("prometheus", map[string]string{
				"cluster": data.Cluster().Name,
			})
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
