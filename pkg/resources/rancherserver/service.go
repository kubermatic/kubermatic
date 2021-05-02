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

package rancherserver

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceCreator creates the service for rancher server
func ServiceCreator(exposeStrategy kubermaticv1.ExposeStrategy) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.RancherServerServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Name = resources.RancherServerServiceName
			s.Labels = resources.BaseAppLabels(resources.RancherStatefulSetName, nil)
			if s.Annotations == nil {
				s.Annotations = map[string]string{}
			}
			if exposeStrategy == kubermaticv1.ExposeStrategyNodePort {
				s.Annotations["nodeport-proxy.k8s.io/expose"] = "true"
				delete(s.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			} else {
				s.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"
				delete(s.Annotations, "nodeport-proxy.k8s.io/expose")
			}
			s.Spec.Selector = resources.BaseAppLabels(resources.RancherStatefulSetName, nil)
			s.Spec.Type = corev1.ServiceTypeNodePort
			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = make([]corev1.ServicePort, 2)
			}

			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromInt(80)
			s.Spec.Ports[0].Name = "http"

			s.Spec.Ports[1].Protocol = corev1.ProtocolTCP
			s.Spec.Ports[1].Port = 443
			s.Spec.Ports[1].TargetPort = intstr.FromInt(443)
			s.Spec.Ports[1].Name = "https"

			return s, nil
		}
	}
}
