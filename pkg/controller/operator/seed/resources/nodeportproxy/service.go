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

package nodeportproxy

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NB: Changing anything in this service can lead to new LoadBalancers being
// created and IPs changing. This must not happen when customers upgrade Kubermatic,
// as all existing kubeconfigs for user clusters would be broken.

const (
	// ServiceName is the name for the created service object.
	ServiceName = "nodeport-proxy"
)

// ServiceCreator bootstraps the nodeport-proxy service object for a seed cluster resource.
func ServiceCreator(seed *kubermaticv1.Seed) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return ServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			// We don't actually manage this service, that is done by the nodeport proxy, we just
			// must make sure that it exists

			s.Spec.Type = corev1.ServiceTypeLoadBalancer
			s.Spec.Selector = map[string]string{
				common.NameLabel: EnvoyDeploymentName,
			}

			// Copy custom annotations if supplied by seed spec.
			if seed.Spec.NodeportProxy.Annotations != nil {
				s.Annotations = seed.Spec.NodeportProxy.Annotations
			}

			// Services need at least one port to be valid, so create it initially.
			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = []corev1.ServicePort{
					{
						Name:       "healthz",
						Port:       EnvoyPort,
						TargetPort: intstr.FromInt(EnvoyPort),
						Protocol:   corev1.ProtocolTCP,
					},
				}
			}

			return s, nil
		}
	}
}
