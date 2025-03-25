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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceReconciler returns the function to reconcile the external OpenVPN service.
func ServiceReconciler(exposeStrategy kubermaticv1.ExposeStrategy) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return resources.OpenVPNServerServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			baseLabels := resources.BaseAppLabels(name, nil)
			kubernetes.EnsureLabels(se, baseLabels)

			if se.Annotations == nil {
				se.Annotations = map[string]string{}
			}
			switch exposeStrategy {
			case kubermaticv1.ExposeStrategyNodePort:
				se.Spec.Type = corev1.ServiceTypeNodePort
				se.Annotations[nodeportproxy.DefaultExposeAnnotationKey] = "true"
				delete(se.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			case kubermaticv1.ExposeStrategyLoadBalancer:
				se.Spec.Type = corev1.ServiceTypeNodePort
				se.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"
				delete(se.Annotations, nodeportproxy.DefaultExposeAnnotationKey)
			case kubermaticv1.ExposeStrategyTunneling:
				se.Spec.Type = corev1.ServiceTypeClusterIP
				se.Annotations[nodeportproxy.DefaultExposeAnnotationKey] = nodeportproxy.TunnelingType.String()
				delete(se.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			default:
				return nil, fmt.Errorf("unsupported expose strategy: %q", exposeStrategy)
			}
			se.Spec.Selector = baseLabels
			if len(se.Spec.Ports) == 0 {
				se.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			se.Spec.Ports[0].Name = "secure"
			se.Spec.Ports[0].Port = 1194
			se.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			se.Spec.Ports[0].TargetPort = intstr.FromInt(1194)

			if exposeStrategy == kubermaticv1.ExposeStrategyTunneling {
				se.Spec.Ports[0].NodePort = 0 // allows switching from other expose strategies
			}

			return se, nil
		}
	}
}
