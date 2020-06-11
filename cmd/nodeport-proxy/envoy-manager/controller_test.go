package main

import (
	"testing"

	"go.uber.org/zap"

	"github.com/go-test/deep"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	envoyv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoycorev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyendpointv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoylistenerv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoytcpfilterv2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
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
		expectedClusters map[string]*envoyv2.Cluster
		expectedListener map[string]*envoyv2.Listener
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
			expectedClusters: map[string]*envoyv2.Cluster{
				"test/my-nodeport-32000": {
					Name:           "test/my-nodeport-32000",
					ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
					ClusterDiscoveryType: &envoyv2.Cluster_Type{
						Type: envoyv2.Cluster_STATIC,
					},
					LbPolicy: envoyv2.Cluster_ROUND_ROBIN,
					LoadAssignment: &envoyv2.ClusterLoadAssignment{
						ClusterName: "test/my-nodeport-32000",
						Endpoints: []*envoyendpointv2.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoyendpointv2.LbEndpoint{
									{
										HostIdentifier: &envoyendpointv2.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv2.Endpoint{
												Address: &envoycorev2.Address{
													Address: &envoycorev2.Address_SocketAddress{
														SocketAddress: &envoycorev2.SocketAddress{
															Protocol: envoycorev2.SocketAddress_TCP,
															Address:  "172.16.0.1",
															PortSpecifier: &envoycorev2.SocketAddress_PortValue{
																PortValue: 8443,
															},
														},
													},
												},
											},
										},
									},
									{
										HostIdentifier: &envoyendpointv2.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv2.Endpoint{
												Address: &envoycorev2.Address{
													Address: &envoycorev2.Address_SocketAddress{
														SocketAddress: &envoycorev2.SocketAddress{
															Protocol: envoycorev2.SocketAddress_TCP,
															Address:  "172.16.0.2",
															PortSpecifier: &envoycorev2.SocketAddress_PortValue{
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
					ClusterDiscoveryType: &envoyv2.Cluster_Type{
						Type: envoyv2.Cluster_STATIC,
					},
					LbPolicy: envoyv2.Cluster_ROUND_ROBIN,
					LoadAssignment: &envoyv2.ClusterLoadAssignment{
						ClusterName: "test/my-nodeport-32001",
						Endpoints: []*envoyendpointv2.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoyendpointv2.LbEndpoint{
									{
										HostIdentifier: &envoyendpointv2.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv2.Endpoint{
												Address: &envoycorev2.Address{
													Address: &envoycorev2.Address_SocketAddress{
														SocketAddress: &envoycorev2.SocketAddress{
															Protocol: envoycorev2.SocketAddress_TCP,
															Address:  "172.16.0.1",
															PortSpecifier: &envoycorev2.SocketAddress_PortValue{
																PortValue: 8080,
															},
														},
													},
												},
											},
										},
									},
									{
										HostIdentifier: &envoyendpointv2.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv2.Endpoint{
												Address: &envoycorev2.Address{
													Address: &envoycorev2.Address_SocketAddress{
														SocketAddress: &envoycorev2.SocketAddress{
															Protocol: envoycorev2.SocketAddress_TCP,
															Address:  "172.16.0.2",
															PortSpecifier: &envoycorev2.SocketAddress_PortValue{
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
			expectedListener: map[string]*envoyv2.Listener{
				"test/my-nodeport-32000": {
					Name: "test/my-nodeport-32000",
					Address: &envoycorev2.Address{
						Address: &envoycorev2.Address_SocketAddress{
							SocketAddress: &envoycorev2.SocketAddress{
								Protocol: envoycorev2.SocketAddress_TCP,
								Address:  "0.0.0.0",
								PortSpecifier: &envoycorev2.SocketAddress_PortValue{
									PortValue: 32000,
								},
							},
						},
					},
					FilterChains: []*envoylistenerv2.FilterChain{
						{
							Filters: []*envoylistenerv2.Filter{
								{
									Name: envoywellknown.TCPProxy,
									ConfigType: &envoylistenerv2.Filter_TypedConfig{
										TypedConfig: marshalMessage(t, &envoytcpfilterv2.TcpProxy{
											StatPrefix: "ingress_tcp",
											ClusterSpecifier: &envoytcpfilterv2.TcpProxy_Cluster{
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
					Address: &envoycorev2.Address{
						Address: &envoycorev2.Address_SocketAddress{
							SocketAddress: &envoycorev2.SocketAddress{
								Protocol: envoycorev2.SocketAddress_TCP,
								Address:  "0.0.0.0",
								PortSpecifier: &envoycorev2.SocketAddress_PortValue{
									PortValue: 32001,
								},
							},
						},
					},
					FilterChains: []*envoylistenerv2.FilterChain{
						{
							Filters: []*envoylistenerv2.Filter{
								{
									Name: envoywellknown.TCPProxy,
									ConfigType: &envoylistenerv2.Filter_TypedConfig{
										TypedConfig: marshalMessage(t, &envoytcpfilterv2.TcpProxy{
											StatPrefix: "ingress_tcp",
											ClusterSpecifier: &envoytcpfilterv2.TcpProxy_Cluster{
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
			expectedClusters: map[string]*envoyv2.Cluster{
				"test/my-nodeport-32001": {
					Name:           "test/my-nodeport-32001",
					ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
					ClusterDiscoveryType: &envoyv2.Cluster_Type{
						Type: envoyv2.Cluster_STATIC,
					},
					LbPolicy: envoyv2.Cluster_ROUND_ROBIN,
					LoadAssignment: &envoyv2.ClusterLoadAssignment{
						ClusterName: "test/my-nodeport-32001",
						Endpoints: []*envoyendpointv2.LocalityLbEndpoints{
							{
								LbEndpoints: []*envoyendpointv2.LbEndpoint{
									{
										HostIdentifier: &envoyendpointv2.LbEndpoint_Endpoint{
											Endpoint: &envoyendpointv2.Endpoint{
												Address: &envoycorev2.Address{
													Address: &envoycorev2.Address_SocketAddress{
														SocketAddress: &envoycorev2.SocketAddress{
															Protocol: envoycorev2.SocketAddress_TCP,
															Address:  "172.16.0.1",
															PortSpecifier: &envoycorev2.SocketAddress_PortValue{
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
			expectedListener: map[string]*envoyv2.Listener{
				"test/my-nodeport-32001": {
					Name: "test/my-nodeport-32001",
					Address: &envoycorev2.Address{
						Address: &envoycorev2.Address_SocketAddress{
							SocketAddress: &envoycorev2.SocketAddress{
								Protocol: envoycorev2.SocketAddress_TCP,
								Address:  "0.0.0.0",
								PortSpecifier: &envoycorev2.SocketAddress_PortValue{
									PortValue: 32001,
								},
							},
						},
					},
					FilterChains: []*envoylistenerv2.FilterChain{
						{
							Filters: []*envoylistenerv2.Filter{
								{
									Name: envoywellknown.TCPProxy,
									ConfigType: &envoylistenerv2.Filter_TypedConfig{
										TypedConfig: marshalMessage(t, &envoytcpfilterv2.TcpProxy{
											StatPrefix: "ingress_tcp",
											ClusterSpecifier: &envoytcpfilterv2.TcpProxy_Cluster{
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
			expectedListener: map[string]*envoyv2.Listener{},
			expectedClusters: map[string]*envoyv2.Cluster{},
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
			expectedListener: map[string]*envoyv2.Listener{},
			expectedClusters: map[string]*envoyv2.Cluster{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log := zap.NewNop().Sugar()
			client := fakectrlruntimeclient.NewFakeClient(test.resources...)
			snapshotCache := envoycache.NewSnapshotCache(true, hasher{}, log)

			c := reconciler{
				Client:              client,
				envoySnapshotCache:  snapshotCache,
				log:                 log,
				lastAppliedSnapshot: envoycache.NewSnapshot("v0.0.0", nil, nil, nil, nil, nil),
			}

			if err := c.sync(); err != nil {
				t.Fatalf("failed to execute controller sync func: %v", err)
			}

			gotClusters := map[string]*envoyv2.Cluster{}
			for name, res := range c.lastAppliedSnapshot.Resources[envoycache.Cluster].Items {
				gotClusters[name] = res.(*envoyv2.Cluster)
			}
			// Delete the admin cluster. We're not going to bother comparing it here, as its a static resource.
			// It would just pollute the testing code
			delete(gotClusters, "service_stats")

			if diff := deep.Equal(gotClusters, test.expectedClusters); diff != nil {
				t.Errorf("Got unexpected clusters. Diff to expected: %v", diff)
			}

			gotListeners := map[string]*envoyv2.Listener{}
			for name, res := range c.lastAppliedSnapshot.Resources[envoycache.Listener].Items {
				gotListeners[name] = res.(*envoyv2.Listener)
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
