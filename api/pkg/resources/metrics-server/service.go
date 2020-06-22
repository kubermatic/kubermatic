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
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceCreator returns the function to reconcile the seed cluster internal metrics-server service
func ServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.MetricsServerServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = resources.MetricsServerServiceName
			labels := resources.BaseAppLabels(name, nil)
			se.Labels = labels

			se.Spec.Selector = labels
			se.Spec.Ports = []corev1.ServicePort{
				{
					Port:       443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(443),
				},
			}

			return se, nil
		}
	}
}
