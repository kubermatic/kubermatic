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
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	operatorresources "k8c.io/kubermatic/v3/pkg/controller/operator/seed/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NB: Changing anything in this service can lead to new LoadBalancers being
// created and IPs changing. This must not happen when admins upgrade Kubermatic,
// as all existing kubeconfigs for user clusters would be broken.

const (
	// ServiceName is the name for the created service object.
	ServiceName = "nodeport-proxy"
)

// ServiceReconciler bootstraps the nodeport-proxy service object for a seed cluster resource.
func ServiceReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return ServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			// We don't actually manage this service, that is done by the nodeport proxy, we just
			// must make sure that it exists

			s.Spec.Type = corev1.ServiceTypeLoadBalancer
			s.Spec.Selector = map[string]string{
				operatorresources.NameLabel: EnvoyDeploymentName,
			}

			if s.Annotations == nil {
				s.Annotations = make(map[string]string)
			}

			// cfg.Spec.NodeportProxy.Annotations is deprecated and should be removed in the future
			// To avoid breaking changes we still copy these values over to the service annotations
			if cfg.Spec.NodeportProxy.Annotations != nil {
				s.Annotations = cfg.Spec.NodeportProxy.Annotations
			}

			// Copy custom annotations specified for the loadBalancer Service. They have a higher precedence then
			// the operatorresources annotations specified in cfg.Spec.NodeportProxy.Annotations, which is deprecated.
			if cfg.Spec.NodeportProxy.Envoy.LoadBalancerService.Annotations != nil {
				for k, v := range cfg.Spec.NodeportProxy.Envoy.LoadBalancerService.Annotations {
					s.Annotations[k] = v
				}
			}

			if cfg.Spec.NodeportProxy.Envoy.LoadBalancerService.SourceRanges != nil {
				for _, cidr := range cfg.Spec.NodeportProxy.Envoy.LoadBalancerService.SourceRanges {
					s.Spec.LoadBalancerSourceRanges = append(s.Spec.LoadBalancerSourceRanges, string(cidr))
				}
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
