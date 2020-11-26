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

package envoymanager

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"

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
	envoywellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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
							DefaultExposeAnnotationKey: "true",
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
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-nodeport",
						Namespace: "test",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Addresses: []corev1.EndpointAddress{
								{IP: "172.16.0.1"},
								{IP: "172.16.0.2"},
							},
							Ports: []corev1.EndpointPort{
								{
									Name:     "http",
									Protocol: corev1.ProtocolTCP,
									Port:     8080,
								},
								{
									Name:     "https",
									Protocol: corev1.ProtocolTCP,
									Port:     8443,
								},
							},
						},
					},
				},
			},
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-nodeport-https": {
					Name:           "test/my-nodeport-https",
					ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
					ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
						Type: envoyclusterv3.Cluster_STATIC,
					},
					LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
					LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
						ClusterName: "test/my-nodeport-https",
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
				"test/my-nodeport-http": {
					Name:           "test/my-nodeport-http",
					ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
					ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
						Type: envoyclusterv3.Cluster_STATIC,
					},
					LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
					LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
						ClusterName: "test/my-nodeport-http",
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
				"test/my-nodeport-https": {
					Name: "test/my-nodeport-https",
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
												Cluster: "test/my-nodeport-https",
											},
										}),
									},
								},
							},
						},
					},
				},
				"test/my-nodeport-http": {
					Name: "test/my-nodeport-http",
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
												Cluster: "test/my-nodeport-http",
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
							DefaultExposeAnnotationKey: "true",
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
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-nodeport",
						Namespace: "test",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Addresses: []corev1.EndpointAddress{
								{IP: "172.16.0.1"},
							},
							NotReadyAddresses: []corev1.EndpointAddress{
								{IP: "172.16.0.2"},
							},
							Ports: []corev1.EndpointPort{
								{
									Name:     "http",
									Protocol: corev1.ProtocolTCP,
									Port:     8080,
								},
							},
						},
					},
				},
			},
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-nodeport-http": {
					Name:           "test/my-nodeport-http",
					ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
					ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
						Type: envoyclusterv3.Cluster_STATIC,
					},
					LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
					LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
						ClusterName: "test/my-nodeport-http",
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
				"test/my-nodeport-http": {
					Name: "test/my-nodeport-http",
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
												Cluster: "test/my-nodeport-http",
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
							DefaultExposeAnnotationKey: "true",
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
			log := zaptest.NewLogger(t).Sugar()
			client := fakectrlruntimeclient.NewFakeClient(test.resources...)
			c, _, _ := NewReconciler(
				context.TODO(),
				log,
				client,
				Options{
					EnvoyNodeName:       "node-name",
					ExposeAnnotationKey: DefaultExposeAnnotationKey,
				},
			)

			if err := c.sync(); err != nil {
				t.Fatalf("failed to execute controller sync func: %v", err)
			}

			gotClusters := map[string]*envoyclusterv3.Cluster{}
			s, _ := c.cache.GetSnapshot(c.EnvoyNodeName)
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

func TestEndpointToService(t *testing.T) {
	tests := []struct {
		name          string
		eps           *corev1.Endpoints
		resources     []runtime.Object
		expectResults []ctrl.Request
	}{
		{
			name:          "No results when matching service is not found",
			eps:           &corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "foo"}},
			expectResults: nil,
		},
		{
			name: "No result when matching service found but not exposed",
			eps:  &corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "foo"}},
			resources: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "foo"},
				},
			},
			expectResults: nil,
		},
		{
			name: "Result expected when exposed matching service is found",
			eps:  &corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}},
			resources: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   "foo",
						Name:        "bar",
						Annotations: map[string]string{DefaultExposeAnnotationKey: "true"},
					},
				},
			},
			expectResults: []ctrl.Request{{
				NamespacedName: types.NamespacedName{Namespace: "foo", Name: "bar"},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := zaptest.NewLogger(t).Sugar()
			client := fakectrlruntimeclient.NewFakeClient(tt.resources...)
			res := (&Reconciler{
				Options: Options{ExposeAnnotationKey: DefaultExposeAnnotationKey},
				Client:  client,
				ctx:     context.TODO(),
				log:     log,
			}).endpointsToService(handler.MapObject{Meta: tt.eps, Object: tt.eps})
			if diff := deep.Equal(res, tt.expectResults); diff != nil {
				t.Errorf("Got unexpected results. Diff to expected: %v", diff)
			}
		})
	}
}

func TestExposeAnnotationPredicate(t *testing.T) {
	tests := []struct {
		name          string
		obj           *corev1.Service
		annotationKey string
		expectAccept  bool
	}{
		{
			name: "Should be accepted when annotation has the good value",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{DefaultExposeAnnotationKey: "true"},
				},
			},
			expectAccept: true,
		},
		{
			name:         "Should be rejected when annotation is not present",
			obj:          &corev1.Service{},
			expectAccept: false,
		},
		{
			name: "Should be rejected when annotation value is wrong",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{DefaultExposeAnnotationKey: "tru"},
				},
			},
			expectAccept: false,
		},
		{
			name: "Should be rejected when annotation value has wrong case",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{DefaultExposeAnnotationKey: "True"},
				},
			},
			expectAccept: false,
		},
		{
			name: "Custom annotation key",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"custom-annotation": "true"},
				},
			},
			annotationKey: "custom-annotation",
			expectAccept:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.annotationKey == "" {
				tt.annotationKey = DefaultExposeAnnotationKey
			}
			p := exposeAnnotationPredicate{annotation: tt.annotationKey, log: zaptest.NewLogger(t).Sugar()}
			if got, exp := p.Create(event.CreateEvent{Meta: tt.obj, Object: tt.obj}), tt.expectAccept; got != exp {
				t.Errorf("expect create accepted %t, but got %t for object: %+v", exp, got, *tt.obj)
			}
			if got, exp := p.Delete(event.DeleteEvent{Meta: tt.obj, Object: tt.obj}), tt.expectAccept; got != exp {
				t.Errorf("expect delete accepted %t, but got %t for object: %+v", exp, got, *tt.obj)
			}
			if got, exp := p.Update(event.UpdateEvent{MetaNew: tt.obj, ObjectNew: tt.obj}), tt.expectAccept; got != exp {
				t.Errorf("expect update accepted %t, but got %t for object: %+v", exp, got, *tt.obj)
			}
			if got, exp := p.Generic(event.GenericEvent{Meta: tt.obj, Object: tt.obj}), tt.expectAccept; got != exp {
				t.Errorf("expect generic accepted %t, but got %t for object: %+v", exp, got, *tt.obj)
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
