/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubernetes

import (
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/utils/ptr"
)

const (
	// serviceName is the name of the kubernetes service in the default namespace.
	serviceName = "kubernetes"

	// endpointSliceName is the name of the endpoint slice for the kubernetes service in the default namespace.
	// To be aligned with upstream endpoint reconcilers we name it as the service itself.
	endpointSliceName = serviceName
)

// EndpointSliceReconciler returns the func to create/update the endpoint slice of the kubernetes service.
func EndpointSliceReconciler(k8sServiceEndpointAddress string, k8sServiceEndpointPort int32) reconciling.NamedEndpointSliceReconcilerFactory {
	return func() (string, reconciling.EndpointSliceReconciler) {
		return endpointSliceName, func(es *discoveryv1.EndpointSlice) (*discoveryv1.EndpointSlice, error) {
			es.AddressType = discoveryv1.AddressTypeIPv4
			es.Labels = map[string]string{
				discoveryv1.LabelServiceName: serviceName,
			}
			es.Endpoints = []discoveryv1.Endpoint{
				{
					Addresses: []string{
						k8sServiceEndpointAddress,
					},
					Conditions: discoveryv1.EndpointConditions{
						Ready: ptr.To(true),
					},
				},
			}
			protoTCP := corev1.ProtocolTCP
			es.Ports = []discoveryv1.EndpointPort{
				{
					Name:     ptr.To("https"),
					Port:     ptr.To[int32](k8sServiceEndpointPort),
					Protocol: &protoTCP,
				},
			}
			return es, nil
		}
	}
}
