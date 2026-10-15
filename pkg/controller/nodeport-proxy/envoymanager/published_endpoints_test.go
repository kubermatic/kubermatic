/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package envoymanager

import (
	"testing"

	"go.uber.org/zap/zaptest"

	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"

	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestPublishedNotReadyEndpoints(t *testing.T) {
	tests := []struct {
		name              string
		published         bool
		oldEndpoint       bool
		expectedAddresses []string
	}{
		{name: "unpublished endpoint"},
		{name: "published endpoint", published: true, expectedAddresses: []string{"172.16.0.2"}},
		{name: "published endpoint alongside ready endpoint", published: true, oldEndpoint: true, expectedAddresses: []string{"172.16.0.1", "172.16.0.2"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := test.NewServiceBuilder(test.NamespacedName{Name: "konnectivity-server", Namespace: "test"}).
				WithServiceType(corev1.ServiceTypeNodePort).
				WithServicePort("secure", 443, 32017, intstr.FromInt(8132), corev1.ProtocolTCP).
				Build()
			endpoints := []discoveryv1.Endpoint{{
				Addresses:  []string{"172.16.0.2"},
				Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(tc.published), Serving: ptr.To(false)},
			}}
			if tc.oldEndpoint {
				endpoints = append([]discoveryv1.Endpoint{{Addresses: []string{"172.16.0.1"}, Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(true), Serving: ptr.To(true)}}}, endpoints...)
			}
			slices := &discoveryv1.EndpointSliceList{Items: []discoveryv1.EndpointSlice{{
				Ports:     []discoveryv1.EndpointPort{{Name: ptr.To("secure"), Port: ptr.To[int32](8132), Protocol: ptr.To(corev1.ProtocolTCP)}},
				Endpoints: endpoints,
			}}}
			sb := newSnapshotBuilder(zaptest.NewLogger(t).Sugar(), portHostMappingFromAnnotation, Options{})
			sb.addService(service, slices, nodeportproxy.ExposeTypes{nodeportproxy.NodePortType: {}})
			gotClusters := map[string]*envoyclusterv3.Cluster{}
			for _, resource := range sb.clusters {
				cluster := resource.(*envoyclusterv3.Cluster)
				gotClusters[cluster.Name] = cluster
			}
			expectedClusters := map[string]*envoyclusterv3.Cluster{}
			expectedListeners := 0
			if len(tc.expectedAddresses) > 0 {
				name := "test/konnectivity-server-secure"
				expectedClusters[name] = makeCluster(t, name, 8132, tc.expectedAddresses...)
				expectedListeners = 1
			}
			if d := diff.ObjectDiff(expectedClusters, gotClusters); d != "" {
				t.Errorf("unexpected clusters:\n%s", d)
			}
			if len(sb.listeners) != expectedListeners {
				t.Errorf("listener count = %d, want %d", len(sb.listeners), expectedListeners)
			}
		})
	}
}
