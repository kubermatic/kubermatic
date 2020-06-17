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
