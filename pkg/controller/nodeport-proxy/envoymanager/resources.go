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
	"net"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	//	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	envoyaccesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyroutev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoylistenerlogv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	envoyhealthv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	envoyhttpconnectionmanagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	envoytcpfilterv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoycachetype "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	envoywellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

const clusterConnectTimeout = 1 * time.Second

const (
	UpgradeType = "CONNECT"
)

// This will be used when the new expose strategy will be fully implemented
//nolint:unused
func makeAccessLog() []*envoyaccesslogv3.AccessLog {
	f := &envoylistenerlogv3.FileAccessLog{
		Path: "/dev/stdout",
	}

	stdoutAccessLog, err := ptypes.MarshalAny(f)
	if err != nil {
		panic(err)
	}

	accessLog := []*envoyaccesslogv3.AccessLog{
		{
			Name: wellknown.FileAccessLog,
			ConfigType: &envoyaccesslogv3.AccessLog_TypedConfig{
				TypedConfig: stdoutAccessLog,
			},
		},
	}
	return accessLog
}

// This will be used when the new expose strategy will be fully implemented
//nolint:unused,deadcode
func makeSNIFilterChain(service *corev1.Service, p portHostMapping) []*envoylistenerv3.FilterChain {
	var sniFilterChains []*envoylistenerv3.FilterChain

	serviceKey := ServiceKey(service)
	for _, servicePort := range service.Spec.Ports {
		servicePortKey := ServicePortKey(serviceKey, &servicePort)

		tcpProxyConfig := &envoytcpfilterv3.TcpProxy{
			StatPrefix: "ingress_tcp",
			ClusterSpecifier: &envoytcpfilterv3.TcpProxy_Cluster{
				Cluster: servicePortKey,
			},
			AccessLog: makeAccessLog(),
		}

		tcpProxyConfigMarshalled, err := ptypes.MarshalAny(tcpProxyConfig)
		if err != nil {
			panic(errors.Wrap(err, "failed to marshal tcpProxyConfig"))
		}

		sniFilterChains = append(sniFilterChains, &envoylistenerv3.FilterChain{
			Filters: []*envoylistenerv3.Filter{
				{
					Name: envoywellknown.TCPProxy,
					ConfigType: &envoylistenerv3.Filter_TypedConfig{
						TypedConfig: tcpProxyConfigMarshalled,
					},
				},
			},
			FilterChainMatch: &envoylistenerv3.FilterChainMatch{
				ServerNames:       []string{p[servicePort.Name]},
				TransportProtocol: "tls",
			},
		})
	}
	return sniFilterChains
}

// This will be used when the new expose strategy will be fully implemented
//nolint:unused
func (r *Reconciler) makeSNIListener() *envoylistenerv3.Listener {
	r.log.Debugf("Using a listener on port %d", r.EnvoySNIListenerPort)

	sniListener := &envoylistenerv3.Listener{
		Name: "sni_listener",
		Address: &envoycorev3.Address{
			Address: &envoycorev3.Address_SocketAddress{
				SocketAddress: &envoycorev3.SocketAddress{
					Protocol: envoycorev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycorev3.SocketAddress_PortValue{
						PortValue: uint32(r.EnvoySNIListenerPort),
					},
				},
			},
		},
	}
	return sniListener
}

// This will be used when the new expose strategy will be fully implemented
//nolint:unused,deadcode
func makeTunnelingVirtualHosts(service *corev1.Service) []*envoyroutev3.VirtualHost {
	var virtualhosts []*envoyroutev3.VirtualHost
	serviceKey := ServiceKey(service)

	for _, servicePort := range service.Spec.Ports {
		servicePortKey := ServicePortKey(serviceKey, &servicePort)

		virtualhosts = append(virtualhosts, &envoyroutev3.VirtualHost{
			Name: servicePortKey,
			Domains: []string{
				fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, servicePort.Port),
			},
			Routes: []*envoyroutev3.Route{
				{
					Match: &envoyroutev3.RouteMatch{
						PathSpecifier: &envoyroutev3.RouteMatch_ConnectMatcher_{
							ConnectMatcher: &envoyroutev3.RouteMatch_ConnectMatcher{},
						},
					},
					Action: &envoyroutev3.Route_Route{
						Route: &envoyroutev3.RouteAction{
							ClusterSpecifier: &envoyroutev3.RouteAction_Cluster{
								Cluster: servicePortKey,
							},
							UpgradeConfigs: []*envoyroutev3.RouteAction_UpgradeConfig{
								{
									UpgradeType:   UpgradeType,
									ConnectConfig: &envoyroutev3.RouteAction_UpgradeConfig_ConnectConfig{},
								},
							},
						},
					},
				},
			},
		})
	}
	return virtualhosts
}

// This will be used when the new expose strategy will be fully implemented
//nolint:unused
func (r *Reconciler) makeTunnelingListener() *envoylistenerv3.Listener {
	httpmanager := &envoyhttpconnectionmanagerv3.HttpConnectionManager{
		CodecType:  envoyhttpconnectionmanagerv3.HttpConnectionManager_HTTP2,
		StatPrefix: "ingress_http",
		RouteSpecifier: &envoyhttpconnectionmanagerv3.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoyroutev3.RouteConfiguration{
				Name: "http2_connect",
			},
		},
		AccessLog: makeAccessLog(),
		HttpFilters: []*envoyhttpconnectionmanagerv3.HttpFilter{
			{
				Name: wellknown.Router,
			},
		},
		Http2ProtocolOptions: &envoycorev3.Http2ProtocolOptions{
			AllowConnect: true,
		},
		UpgradeConfigs: []*envoyhttpconnectionmanagerv3.HttpConnectionManager_UpgradeConfig{
			{
				UpgradeType: UpgradeType,
			},
		},
	}

	HTTPManagerConfigMarshalled, err := ptypes.MarshalAny(httpmanager)
	if err != nil {
		panic(errors.Wrap(err, "failed to marshal HTTP Connection Manager"))
	}

	r.log.Debugf("Using a listener on port %d", r.EnvoyHTTP2ConnectListenerPort)

	tunnelingListener := &envoylistenerv3.Listener{
		Name: "http2connect_listener",
		Address: &envoycorev3.Address{
			Address: &envoycorev3.Address_SocketAddress{
				SocketAddress: &envoycorev3.SocketAddress{
					Protocol: envoycorev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycorev3.SocketAddress_PortValue{
						PortValue: uint32(r.EnvoyHTTP2ConnectListenerPort),
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
							TypedConfig: HTTPManagerConfigMarshalled,
						},
					},
				},
			},
		},
	}
	return tunnelingListener
}

func (r *Reconciler) makeClusters(service *corev1.Service, endpoints *corev1.Endpoints) (clusters []envoycachetype.Resource) {
	serviceKey := ServiceKey(service)
	for _, servicePort := range service.Spec.Ports {
		servicePortKey := ServicePortKey(serviceKey, &servicePort)
		endpoints := r.getEndpoints(service, &servicePort, corev1.ProtocolTCP, endpoints)

		// Must be sorted, otherwise we get into trouble when doing the snapshot diff later
		sort.Slice(endpoints, func(i, j int) bool {
			addrI := endpoints[i].HostIdentifier.(*envoyendpointv3.LbEndpoint_Endpoint).Endpoint.Address.Address.(*envoycorev3.Address_SocketAddress).SocketAddress.Address
			addrJ := endpoints[j].HostIdentifier.(*envoyendpointv3.LbEndpoint_Endpoint).Endpoint.Address.Address.(*envoycorev3.Address_SocketAddress).SocketAddress.Address
			return addrI < addrJ
		})

		cluster := &envoyclusterv3.Cluster{
			Name:           servicePortKey,
			ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
			ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
				Type: envoyclusterv3.Cluster_STATIC,
			},
			LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
			LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
				ClusterName: servicePortKey,
				Endpoints: []*envoyendpointv3.LocalityLbEndpoints{
					{
						LbEndpoints: endpoints,
					},
				},
			},
		}
		clusters = append(clusters, cluster)
	}
	return
}

func (r *Reconciler) makeListenersForNodePortService(service *corev1.Service) (listeners []envoycachetype.Resource) {
	serviceKey := ServiceKey(service)
	for _, servicePort := range service.Spec.Ports {
		servicePortKey := ServicePortKey(serviceKey, &servicePort)

		tcpProxyConfig := &envoytcpfilterv3.TcpProxy{
			StatPrefix: "ingress_tcp",
			ClusterSpecifier: &envoytcpfilterv3.TcpProxy_Cluster{
				Cluster: servicePortKey,
			},
		}

		tcpProxyConfigMarshalled, err := ptypes.MarshalAny(tcpProxyConfig)
		if err != nil {
			panic(errors.Wrap(err, "failed to marshal tcpProxyConfig"))
		}

		r.log.Debugf("Using a listener on port %d", servicePort.NodePort)

		listener := &envoylistenerv3.Listener{
			Name: servicePortKey,
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

// getEndpoints returns a slice of LbEndpoint pointers for a given
// service/target port combination.
// Based on:
// https://github.com/kubernetes/ingress-nginx/blob/decc1346dd956a7f3edfc23c2547abbc75598e36/internal/ingress/controller/endpoints.go#L35
func (r *Reconciler) getEndpoints(s *corev1.Service, port *corev1.ServicePort, proto corev1.Protocol,
	eps *corev1.Endpoints) []*envoyendpointv3.LbEndpoint {

	var upsServers []*envoyendpointv3.LbEndpoint

	if s == nil || port == nil {
		return upsServers
	}

	// using a map avoids duplicated upstream servers when the service
	// contains multiple port definitions sharing the same targetport
	processedUpstreamServers := make(map[string]struct{})

	svcKey := ServiceKey(s)
	serviceLog := r.log.With("service", svcKey)

	for _, ss := range eps.Subsets {
		for _, epPort := range ss.Ports {

			if !reflect.DeepEqual(epPort.Protocol, proto) {
				continue
			}

			var targetPort int32

			if port.Name == "" {
				// port.Name is optional if there is only one port
				targetPort = epPort.Port
			} else if port.Name == epPort.Name {
				targetPort = epPort.Port
			}

			if targetPort <= 0 {
				continue
			}

			for _, epAddress := range ss.Addresses {
				ep := net.JoinHostPort(epAddress.IP, strconv.Itoa(int(targetPort)))
				if _, exists := processedUpstreamServers[ep]; exists {
					continue
				}
				ups := &envoyendpointv3.LbEndpoint{
					HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
						Endpoint: &envoyendpointv3.Endpoint{
							Address: &envoycorev3.Address{
								Address: &envoycorev3.Address_SocketAddress{
									SocketAddress: &envoycorev3.SocketAddress{
										Protocol: envoycorev3.SocketAddress_TCP,
										Address:  epAddress.IP,
										PortSpecifier: &envoycorev3.SocketAddress_PortValue{
											PortValue: uint32(targetPort),
										},
									},
								},
							},
						},
					},
				}
				upsServers = append(upsServers, ups)
				processedUpstreamServers[ep] = struct{}{}
			}
		}
	}

	serviceLog.Debugw("Endpoints found", "lb-endpoints", upsServers)
	return upsServers
}

func newSnapshot(version string, clusters, listeners []envoycachetype.Resource) envoycachev3.Snapshot {
	return envoycachev3.NewSnapshot(
		version,
		nil,       // endpoints
		clusters,  // clusters
		nil,       // routes
		listeners, // listeners
		nil,       // runtimes
		nil,       // secrets
	)
}
