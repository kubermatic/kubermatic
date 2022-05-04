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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	"k8s.io/utils/pointer"
)

const (
	// serviceName is the name of the kubernetes service in the default namespace.
	serviceName = "kubernetes"

	// endpointSliceName is the name of the endpoint slice for the kubernetes service in the default namespace.
	// To be aligned with upstream endpoint reconcilers we name it as the service itself.
	endpointSliceName = serviceName
)

// EndpointsCreator returns the func to create/update the endpoints of the kubernetes service.
func EndpointsCreator(clusterAddress *kubermaticv1.ClusterAddress) reconciling.NamedEndpointsCreatorGetter {
	return func() (string, reconciling.EndpointsCreator) {
		return serviceName, func(ep *corev1.Endpoints) (*corev1.Endpoints, error) {
			// our controller is reconciling the endpoint slice, do not mirror with EndpointSliceMirroring controller
			ep.Labels = map[string]string{
				discovery.LabelSkipMirror: "true",
			}
			ep.Subsets = []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: clusterAddress.IP,
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Name:     "https",
							Port:     clusterAddress.Port,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			}
			return ep, nil
		}
	}
}

// EndpointSliceCreator returns the func to create/update the endpoint slice of the kubernetes service.
func EndpointSliceCreator(clusterAddress *kubermaticv1.ClusterAddress) reconciling.NamedEndpointSliceCreatorGetter {
	return func() (string, reconciling.EndpointSliceCreator) {
		return endpointSliceName, func(es *discovery.EndpointSlice) (*discovery.EndpointSlice, error) {
			es.AddressType = discovery.AddressTypeIPv4
			es.Labels = map[string]string{
				discovery.LabelServiceName: serviceName,
			}
			es.Endpoints = []discovery.Endpoint{
				{
					Addresses: []string{
						clusterAddress.IP,
					},
					Conditions: discovery.EndpointConditions{
						Ready: pointer.BoolPtr(true),
					},
				},
			}
			protoTCP := corev1.ProtocolTCP
			es.Ports = []discovery.EndpointPort{
				{
					Name:     pointer.String("https"),
					Port:     pointer.Int32Ptr(clusterAddress.Port),
					Protocol: &protoTCP,
				},
			}
			return es, nil
		}
	}
}
