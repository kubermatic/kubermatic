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

package gatekeeper

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceCreator returns the function to reconcile the gatekeeper service
func ServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.GatekeeperWebhookServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = resources.GatekeeperWebhookServiceName
			labels := resources.BaseAppLabels(controllerName, map[string]string{"gatekeeper.sh/system": "yes"})
			se.Labels = labels

			se.Spec.Type = corev1.ServiceTypeClusterIP
			se.Spec.Selector = gatekeeperControllerLabels
			se.Spec.Ports = []corev1.ServicePort{
				{
					Port:       443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8443),
				},
			}

			return se, nil
		}
	}
}
