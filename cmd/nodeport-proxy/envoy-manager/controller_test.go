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

package main

import (
	"testing"

	"go.uber.org/zap"

	"github.com/go-test/deep"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoytcpfilterv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoycachetype "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoywellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSync(t *testing.T) {
	tests := []struct {
		name             string
		resources        []runtime.Object
		expectedClusters map[string]*envoyclusterv3.Cluster
		expectedListener map[string]*envoylistenerv3.Listener
	}{
		{
			name: "2-ports-2-pods-named-and-non-named-ports",
			resources: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-nodeport",
						Namespace: "test",
						Annotations: map[string]string{
							exposeAnnotationKey: "true",
						},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
						Ports: []corev1.ServicePort{
							// Test if we can proxy to named ports
							{
								Name:       "http",
								TargetPort: intstr.FromString("http"),
								NodePort:   32001,
								Protocol:   corev1.ProtocolTCP,
								Port:       80,
							},
							// Test if we can proxy to int ports
							{
								Name:       "https",
								TargetPort: intstr.FromInt(8443),
								NodePort:   32000,
								Protocol:   corev1.ProtocolTCP,
								Port:       443,
							},
						},
						Selector: map[string]string{
							"foo": "bar",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "webservice",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 8080,
									},
									{
										Name:          "https",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 8443,
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						PodIP: "172.16.0.1",
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "test",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "webservice",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 8080,
									},
									{
										Name:          "https",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 8443,
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						PodIP: "172.16.0.2",
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-nodeport-32000": {
					Name:           "test/my-nodeport-32000",
					ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
					ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
						Type: envoyclusterv3.Cluster_STATIC,
					},
					LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
					LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
						ClusterName: "test/my-nodeport-32000",
						Endpoints: []*envoyendpointv3.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoyendpointv3.LbEndpoint{
									{
										HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv3.Endpoint{
												Address: &envoycorev3.Address{
													Address: &envoycorev3.Address_SocketAddress{
														SocketAddress: &envoycorev3.SocketAddress{
															Protocol: envoycorev3.SocketAddress_TCP,
															Address:  "172.16.0.1",
															PortSpecifier: &envoycorev3.SocketAddress_PortValue{
																PortValue: 8443,
															},
														},
													},
												},
											},
										},
									},
									{
										HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv3.Endpoint{
												Address: &envoycorev3.Address{
													Address: &envoycorev3.Address_SocketAddress{
														SocketAddress: &envoycorev3.SocketAddress{
															Protocol: envoycorev3.SocketAddress_TCP,
															Address:  "172.16.0.2",
															PortSpecifier: &envoycorev3.SocketAddress_PortValue{
																PortValue: 8443,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				"test/my-nodeport-32001": {
					Name:           "test/my-nodeport-32001",
					ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
					ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
						Type: envoyclusterv3.Cluster_STATIC,
					},
					LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
					LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
						ClusterName: "test/my-nodeport-32001",
						Endpoints: []*envoyendpointv3.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoyendpointv3.LbEndpoint{
									{
										HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv3.Endpoint{
												Address: &envoycorev3.Address{
													Address: &envoycorev3.Address_SocketAddress{
														SocketAddress: &envoycorev3.SocketAddress{
															Protocol: envoycorev3.SocketAddress_TCP,
															Address:  "172.16.0.1",
															PortSpecifier: &envoycorev3.SocketAddress_PortValue{
																PortValue: 8080,
															},
														},
													},
												},
											},
										},
									},
									{
										HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv3.Endpoint{
												Address: &envoycorev3.Address{
													Address: &envoycorev3.Address_SocketAddress{
														SocketAddress: &envoycorev3.SocketAddress{
															Protocol: envoycorev3.SocketAddress_TCP,
															Address:  "172.16.0.2",
															PortSpecifier: &envoycorev3.SocketAddress_PortValue{
																PortValue: 8080,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"test/my-nodeport-32000": {
					Name: "test/my-nodeport-32000",
					Address: &envoycorev3.Address{
						Address: &envoycorev3.Address_SocketAddress{
							SocketAddress: &envoycorev3.SocketAddress{
								Protocol: envoycorev3.SocketAddress_TCP,
								Address:  "0.0.0.0",
								PortSpecifier: &envoycorev3.SocketAddress_PortValue{
									PortValue: 32000,
								},
							},
						},
					},
					FilterChains: []*envoylistenerv3.FilterChain{
						{
							Filters: []*envoylistenerv3.Filter{
								{
									Name: envoywellknown.TCPProxy,
									ConfigType: &envoylistenerv3.Filter_TypedConfig{
										TypedConfig: marshalMessage(t, &envoytcpfilterv3.TcpProxy{
											StatPrefix: "ingress_tcp",
											ClusterSpecifier: &envoytcpfilterv3.TcpProxy_Cluster{
												Cluster: "test/my-nodeport-32000",
											},
										}),
									},
								},
							},
						},
					},
				},
				"test/my-nodeport-32001": {
					Name: "test/my-nodeport-32001",
					Address: &envoycorev3.Address{
						Address: &envoycorev3.Address_SocketAddress{
							SocketAddress: &envoycorev3.SocketAddress{
								Protocol: envoycorev3.SocketAddress_TCP,
								Address:  "0.0.0.0",
								PortSpecifier: &envoycorev3.SocketAddress_PortValue{
									PortValue: 32001,
								},
							},
						},
					},
					FilterChains: []*envoylistenerv3.FilterChain{
						{
							Filters: []*envoylistenerv3.Filter{
								{
									Name: envoywellknown.TCPProxy,
									ConfigType: &envoylistenerv3.Filter_TypedConfig{
										TypedConfig: marshalMessage(t, &envoytcpfilterv3.TcpProxy{
											StatPrefix: "ingress_tcp",
											ClusterSpecifier: &envoytcpfilterv3.TcpProxy_Cluster{
												Cluster: "test/my-nodeport-32001",
											},
										}),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "1-port-2-pods-one-unhealthy",
			resources: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-nodeport",
						Namespace: "test",
						Annotations: map[string]string{
							exposeAnnotationKey: "true",
						},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
						Ports: []corev1.ServicePort{
							{
								Name:       "http",
								TargetPort: intstr.FromString("http"),
								NodePort:   32001,
								Protocol:   corev1.ProtocolTCP,
								Port:       80,
							},
						},
						Selector: map[string]string{
							"foo": "bar",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "webservice",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 8080,
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						PodIP: "172.16.0.1",
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "test",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "webservice",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 8080,
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						PodIP: "172.16.0.2",
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionFalse,
							},
						},
					},
				},
			},
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-nodeport-32001": {
					Name:           "test/my-nodeport-32001",
					ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
					ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
						Type: envoyclusterv3.Cluster_STATIC,
					},
					LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
					LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
						ClusterName: "test/my-nodeport-32001",
						Endpoints: []*envoyendpointv3.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoyendpointv3.LbEndpoint{
									{
										HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv3.Endpoint{
												Address: &envoycorev3.Address{
													Address: &envoycorev3.Address_SocketAddress{
														SocketAddress: &envoycorev3.SocketAddress{
															Protocol: envoycorev3.SocketAddress_TCP,
															Address:  "172.16.0.1",
															PortSpecifier: &envoycorev3.SocketAddress_PortValue{
																PortValue: 8080,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"test/my-nodeport-32001": {
					Name: "test/my-nodeport-32001",
					Address: &envoycorev3.Address{
						Address: &envoycorev3.Address_SocketAddress{
							SocketAddress: &envoycorev3.SocketAddress{
								Protocol: envoycorev3.SocketAddress_TCP,
								Address:  "0.0.0.0",
								PortSpecifier: &envoycorev3.SocketAddress_PortValue{
									PortValue: 32001,
								},
							},
						},
					},
					FilterChains: []*envoylistenerv3.FilterChain{
						{
							Filters: []*envoylistenerv3.Filter{
								{
									Name: envoywellknown.TCPProxy,
									ConfigType: &envoylistenerv3.Filter_TypedConfig{
										TypedConfig: marshalMessage(t, &envoytcpfilterv3.TcpProxy{
											StatPrefix: "ingress_tcp",
											ClusterSpecifier: &envoytcpfilterv3.TcpProxy_Cluster{
												Cluster: "test/my-nodeport-32001",
											},
										}),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "1-port-service-without-annotation",
			resources: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-nodeport",
						Namespace: "test",
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
						Ports: []corev1.ServicePort{
							{
								Name:       "http",
								TargetPort: intstr.FromString("http"),
								NodePort:   32001,
								Protocol:   corev1.ProtocolTCP,
								Port:       80,
							},
						},
						Selector: map[string]string{
							"foo": "bar",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "webservice",
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 8080,
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						PodIP: "172.16.0.1",
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			expectedListener: map[string]*envoylistenerv3.Listener{},
			expectedClusters: map[string]*envoyclusterv3.Cluster{},
		},
		{
			name: "1-port-service-without-pods",
			resources: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-nodeport",
						Namespace: "test",
						Annotations: map[string]string{
							exposeAnnotationKey: "true",
						},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort,
						Ports: []corev1.ServicePort{
							{
								Name:       "http",
								TargetPort: intstr.FromString("http"),
								NodePort:   32001,
								Protocol:   corev1.ProtocolTCP,
								Port:       80,
							},
						},
						Selector: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			expectedListener: map[string]*envoylistenerv3.Listener{},
			expectedClusters: map[string]*envoyclusterv3.Cluster{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log := zap.NewNop().Sugar()
			client := fakectrlruntimeclient.NewFakeClient(test.resources...)
			snapshotCache := envoycachev3.NewSnapshotCache(true, hasher{}, log)

			c := reconciler{
				Client:             client,
				envoySnapshotCache: snapshotCache,
				log:                log,
			}

			if err := c.sync(); err != nil {
				t.Fatalf("failed to execute controller sync func: %v", err)
			}

			gotClusters := map[string]*envoyclusterv3.Cluster{}
			s, _ := c.envoySnapshotCache.GetSnapshot(envoyNodeName)
			for name, res := range s.Resources[envoycachetype.Cluster].Items {
				gotClusters[name] = res.(*envoyclusterv3.Cluster)
			}
			// Delete the admin cluster. We're not going to bother comparing it here, as its a static resource.
			// It would just pollute the testing code
			delete(gotClusters, "service_stats")

			if diff := deep.Equal(gotClusters, test.expectedClusters); diff != nil {
				t.Errorf("Got unexpected clusters. Diff to expected: %v", diff)
			}

			gotListeners := map[string]*envoylistenerv3.Listener{}
			for name, res := range s.Resources[envoycachetype.Listener].Items {
				gotListeners[name] = res.(*envoylistenerv3.Listener)
			}
			delete(gotListeners, "service_stats")

			if diff := deep.Equal(gotListeners, test.expectedListener); diff != nil {
				t.Errorf("Got unexpected listeners. Diff to expected: %v", diff)
			}
		})
	}
}

func marshalMessage(t *testing.T, msg proto.Message) *any.Any {
	marshalled, err := ptypes.MarshalAny(msg)
	if err != nil {
		t.Fatalf("failed to marshal from message to any: %v", err)
	}

	return marshalled
}
