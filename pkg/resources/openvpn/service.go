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

package openvpn

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/controller/nodeport-proxy/envoymanager"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceCreator returns the function to reconcile the external OpenVPN service
func ServiceCreator(exposeStrategy kubermaticv1.ExposeStrategy) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.OpenVPNServerServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = resources.OpenVPNServerServiceName
			se.Labels = resources.BaseAppLabels(name, nil)

			if se.Annotations == nil {
				se.Annotations = map[string]string{}
			}
			switch exposeStrategy {
			case kubermaticv1.ExposeStrategyNodePort:
				se.Spec.Type = corev1.ServiceTypeNodePort
				se.Annotations[envoymanager.DefaultExposeAnnotationKey] = "true"
				delete(se.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			case kubermaticv1.ExposeStrategyLoadBalancer:
				se.Spec.Type = corev1.ServiceTypeNodePort
				se.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"
				delete(se.Annotations, envoymanager.DefaultExposeAnnotationKey)
			case kubermaticv1.ExposeStrategyTunneling:
				se.Spec.Type = corev1.ServiceTypeClusterIP
				se.Annotations[envoymanager.DefaultExposeAnnotationKey] = envoymanager.TunnelingType.String()
				delete(se.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			default:
				return nil, fmt.Errorf("unsupported expose strategy: %q", exposeStrategy)
			}
			se.Spec.Selector = map[string]string{
				resources.AppLabelKey: name,
			}
			if len(se.Spec.Ports) == 0 {
				se.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			se.Spec.Ports[0].Name = "secure"
			se.Spec.Ports[0].Port = 1194
			se.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			se.Spec.Ports[0].TargetPort = intstr.FromInt(1194)

			return se, nil
		}
	}
}
