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

package konnectivity

import (
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceReconciler returns function to create konnectivity proxy service.
func ServiceReconciler(exposeStrategy kubermaticv1.ExposeStrategy, externalURL string) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return resources.KonnectivityProxyServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			// because konnectivity proxy runs in sidecar in apiserver pod
			se.Spec.Selector = resources.BaseAppLabels(resources.ApiserverDeploymentName, nil)

			if se.Annotations == nil {
				se.Annotations = make(map[string]string)
			}

			switch exposeStrategy {
			case kubermaticv1.ExposeStrategyNodePort:
				se.Spec.Type = corev1.ServiceTypeNodePort
				se.Annotations[nodeportproxy.DefaultExposeAnnotationKey] = nodeportproxy.NodePortType.String()
				delete(se.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			case kubermaticv1.ExposeStrategyLoadBalancer:
				se.Spec.Type = corev1.ServiceTypeNodePort
				se.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"
				delete(se.Annotations, nodeportproxy.DefaultExposeAnnotationKey)
			case kubermaticv1.ExposeStrategyTunneling:
				se.Spec.Type = corev1.ServiceTypeClusterIP
				se.Annotations[nodeportproxy.DefaultExposeAnnotationKey] = strings.Join([]string{nodeportproxy.SNIType.String(), nodeportproxy.TunnelingType.String()}, ",")
				se.Annotations[nodeportproxy.PortHostMappingAnnotationKey] = fmt.Sprintf(`{"secure": %q}`, "konnectivity-server."+externalURL)
				delete(se.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			default:
				return nil, fmt.Errorf("unsupported expose strategy: %q", exposeStrategy)
			}

			if len(se.Spec.Ports) == 0 {
				se.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			const port = 8132

			se.Spec.Ports[0].Name = "secure"
			se.Spec.Ports[0].Port = 443
			se.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			se.Spec.Ports[0].TargetPort = intstr.FromInt(port)

			if exposeStrategy == kubermaticv1.ExposeStrategyTunneling {
				se.Spec.Ports[0].NodePort = 0
			}

			return se, nil
		}
	}
}
