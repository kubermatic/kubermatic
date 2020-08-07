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

package apiserver

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceCreator returns the function to reconcile the external API server service
func ServiceCreator(exposeStrategy corev1.ServiceType) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.ApiserverServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			// Always set it to NodePort. Even when using exposeStrategy==LoadBalancer, we create
			// one LoadBalancer for two Services (APIServer and openVPN) and use the nodePortProxy in
			// namespaced mode to redirect the traffic to the right service depending on its port.
			// We use a nodePort Service because that gives us a concurrency-safe allocation mechanism
			// for a unique port
			se.Spec.Type = corev1.ServiceTypeNodePort
			if se.Annotations == nil {
				se.Annotations = map[string]string{}
			}
			if exposeStrategy != corev1.ServiceTypeNodePort && exposeStrategy != corev1.ServiceTypeLoadBalancer {
				return nil, fmt.Errorf("exposeStrategy on the cluster must be one of `NodePort` or `LoadBalancer, got %q`", exposeStrategy)
			}
			if exposeStrategy == corev1.ServiceTypeNodePort {
				se.Annotations["nodeport-proxy.k8s.io/expose"] = "true"
				delete(se.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			} else {
				se.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"
				delete(se.Annotations, "nodeport-proxy.k8s.io/expose")
			}

			se.Spec.Selector = map[string]string{
				resources.AppLabelKey: name,
			}

			if len(se.Spec.Ports) == 0 {
				se.Spec.Ports = []corev1.ServicePort{
					{
						Name:       "secure",
						Port:       443,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(resources.APIServerSecurePort),
					},
				}

				return se, nil
			}

			se.Spec.Ports[0].Name = "secure"
			se.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			se.Spec.Ports[0].Port = 443
			// We assign the target port the same value as the NodePort port.
			// The reason is that we need  both access the apiserver using
			// this service (i.e. from seed cluster) and from the kubernetes
			// nodeport service in the default namespace of the user cluster.
			se.Spec.Ports[0].TargetPort = intstr.FromInt(int(se.Spec.Ports[0].NodePort))

			return se, nil
		}
	}
}
