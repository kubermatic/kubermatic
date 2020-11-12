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
	"fmt"
	"sort"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyroutev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoyhealthv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	envoyhttpconnectionmanagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	envoytcpfilterv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoycachetype "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoywellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

func (r *Reconciler) makeListenersAndClustersForService(service *corev1.Service, pods []*corev1.Pod) (listeners []envoycachetype.Resource, clusters []envoycachetype.Resource) {
	serviceKey := ServiceKey(service)
	for _, servicePort := range service.Spec.Ports {
		serviceNodePortName := fmt.Sprintf("%s-%d", serviceKey, servicePort.NodePort)
		servicePortLog := r.Log.With("port", servicePort.NodePort)

		var endpoints []*envoyendpointv3.LbEndpoint
		for _, pod := range pods {
			podLog := servicePortLog.With("pod", pod.Name, "namespace", pod.Namespace, "service", service.Name)

			// Get the port on the pod, the NodePort Service port is pointing to
			podPort := getMatchingPodPort(servicePort, pod)
			if podPort == 0 {
				podLog.Debug("Skipping pod for service port: the service port does not match to any of the pods containers")
				continue
			}

			podLog.Debug("Using pod as backend for service")

			// Cluster endpoints
			endpoints = append(endpoints, &envoyendpointv3.LbEndpoint{
				HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
					Endpoint: &envoyendpointv3.Endpoint{
						Address: &envoycorev3.Address{
							Address: &envoycorev3.Address_SocketAddress{
								SocketAddress: &envoycorev3.SocketAddress{
									Protocol: envoycorev3.SocketAddress_TCP,
									Address:  pod.Status.PodIP,
									PortSpecifier: &envoycorev3.SocketAddress_PortValue{
										PortValue: uint32(podPort),
									},
								},
							},
						},
					},
				},
			})
		}

		// Must be sorted, otherwise we get into trouble when doing the snapshot diff later
		sort.Slice(endpoints, func(i, j int) bool {
			addrI := endpoints[i].HostIdentifier.(*envoyendpointv3.LbEndpoint_Endpoint).Endpoint.Address.Address.(*envoycorev3.Address_SocketAddress).SocketAddress.Address
			addrJ := endpoints[j].HostIdentifier.(*envoyendpointv3.LbEndpoint_Endpoint).Endpoint.Address.Address.(*envoycorev3.Address_SocketAddress).SocketAddress.Address
			return addrI < addrJ
		})

		cluster := &envoyclusterv3.Cluster{
			Name:           serviceNodePortName,
			ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
			ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
				Type: envoyclusterv3.Cluster_STATIC,
			},
			LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
			LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
				ClusterName: serviceNodePortName,
				Endpoints: []*envoyendpointv3.LocalityLbEndpoints{
					{
						LbEndpoints: endpoints,
					},
				},
			},
		}
		clusters = append(clusters, cluster)

		tcpProxyConfig := &envoytcpfilterv3.TcpProxy{
			StatPrefix: "ingress_tcp",
			ClusterSpecifier: &envoytcpfilterv3.TcpProxy_Cluster{
				Cluster: serviceNodePortName,
			},
		}

		tcpProxyConfigMarshalled, err := ptypes.MarshalAny(tcpProxyConfig)
		if err != nil {
			panic(errors.Wrap(err, "failed to marshal tcpProxyConfig"))
		}

		r.Log.Debugf("Using a listener on port %d", servicePort.NodePort)

		listener := &envoylistenerv3.Listener{
			Name: serviceNodePortName,
			Address: &envoycorev3.Address{
				Address: &envoycorev3.Address_SocketAddress{
					SocketAddress: &envoycorev3.SocketAddress{
						Protocol: envoycorev3.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &envoycorev3.SocketAddress_PortValue{
							PortValue: uint32(servicePort.NodePort),
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
								TypedConfig: tcpProxyConfigMarshalled,
							},
						},
					},
				},
			},
		}
		listeners = append(listeners, listener)
	}
	return
}

func (r *Reconciler) makeInitialResources() (listeners []envoycachetype.Resource, clusters []envoycachetype.Resource) {
	adminCluster := &envoyclusterv3.Cluster{
		Name:           "service_stats",
		ConnectTimeout: ptypes.DurationProto(50 * time.Millisecond),
		ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
			Type: envoyclusterv3.Cluster_STATIC,
		},
		LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
			ClusterName: "service_stats",
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
												Address:  "127.0.0.1",
												PortSpecifier: &envoycorev3.SocketAddress_PortValue{
													PortValue: uint32(r.EnvoyAdminPort),
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
	}
	clusters = append(clusters, adminCluster)

	healthCheck := &envoyhealthv3.HealthCheck{
		PassThroughMode: &wrappers.BoolValue{Value: false},
		Headers: []*envoyroutev3.HeaderMatcher{
			{
				Name: ":path",
				HeaderMatchSpecifier: &envoyroutev3.HeaderMatcher_ExactMatch{
					ExactMatch: "/healthz",
				},
			},
		},
	}

	healthCheckMarshalled, err := ptypes.MarshalAny(healthCheck)
	if err != nil {
		// panic as this either never occurs or cannot recover
		panic(errors.Wrap(err, "failed to marshal HealthCheck"))
	}

	httpConnectionManager := &envoyhttpconnectionmanagerv3.HttpConnectionManager{
		CodecType:  envoyhttpconnectionmanagerv3.HttpConnectionManager_AUTO,
		StatPrefix: "service_stats",
		RouteSpecifier: &envoyhttpconnectionmanagerv3.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoyroutev3.RouteConfiguration{
				VirtualHosts: []*envoyroutev3.VirtualHost{
					{
						Name:    "backend",
						Domains: []string{"*"},
						Routes: []*envoyroutev3.Route{
							{
								Match: &envoyroutev3.RouteMatch{
									PathSpecifier: &envoyroutev3.RouteMatch_Prefix{
										Prefix: "/stats",
									},
								},
								Action: &envoyroutev3.Route_Route{
									Route: &envoyroutev3.RouteAction{
										ClusterSpecifier: &envoyroutev3.RouteAction_Cluster{
											Cluster: "service_stats",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		HttpFilters: []*envoyhttpconnectionmanagerv3.HttpFilter{
			{
				Name: envoywellknown.HealthCheck,
				ConfigType: &envoyhttpconnectionmanagerv3.HttpFilter_TypedConfig{
					TypedConfig: healthCheckMarshalled,
				},
			},
			{
				Name: envoywellknown.Router,
			},
		},
	}

	httpConnectionManagerMarshalled, err := ptypes.MarshalAny(httpConnectionManager)
	if err != nil {
		panic(errors.Wrap(err, "failed to marshal HTTPConnectionManager"))
	}

	listener := &envoylistenerv3.Listener{
		Name: "service_stats",
		Address: &envoycorev3.Address{
			Address: &envoycorev3.Address_SocketAddress{
				SocketAddress: &envoycorev3.SocketAddress{
					Protocol: envoycorev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycorev3.SocketAddress_PortValue{
						PortValue: uint32(r.EnvoyStatsPort),
					},
				},
			},
		},
		FilterChains: []*envoylistenerv3.FilterChain{
			{
				Filters: []*envoylistenerv3.Filter{
					{
						Name: envoywellknown.HTTPConnectionManager,
						ConfigType: &envoylistenerv3.Filter_TypedConfig{
							TypedConfig: httpConnectionManagerMarshalled,
						},
					},
				},
			},
		},
	}
	listeners = append(listeners, listener)

	return listeners, clusters
}
