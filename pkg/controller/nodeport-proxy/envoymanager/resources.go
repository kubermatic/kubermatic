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
	"sort"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	envoyaccesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyroutev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoylistenerlogv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	envoyhealthv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	envoyrouterv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	envoytlsinspectorv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
	envoyhttpconnectionmanagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	envoytcpfilterv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoymatcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoycachetype "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoyresourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	envoywellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

const clusterConnectTimeout = 1 * time.Second

const (
	UpgradeType = "CONNECT"
)

// portHostMappingGetter returns the portHostMapping for the given Service or
// an error.
type portHostMappingGetter func(*corev1.Service) (portHostMapping, error)

// snapshotBuilder builds an Envoy configuration Snapshot.
// Current implementation is not thread-safe.
type snapshotBuilder struct {
	Options
	log                   *zap.SugaredLogger
	portHostMappingGetter portHostMappingGetter

	// book-keeping
	fcs       []*envoylistenerv3.FilterChain
	vhs       []*envoyroutev3.VirtualHost
	listeners []envoycachetype.Resource
	clusters  []envoycachetype.Resource
	// keeps a mapping between hostnames and service keys
	hostnameToService map[string]types.NamespacedName
}

func newSnapshotBuilder(log *zap.SugaredLogger, portHostMappingGetter portHostMappingGetter, opts Options) *snapshotBuilder {
	sb := snapshotBuilder{
		log:                   log.With("component", "snapshotBuilder"),
		Options:               opts,
		portHostMappingGetter: portHostMappingGetter,
		hostnameToService:     map[string]types.NamespacedName{},
	}
	return &sb
}

// addService adds a Service to the builder with the associated service types.
func (sb *snapshotBuilder) addService(svc *corev1.Service, epSlices *discoveryv1.EndpointSliceList, expTypes nodeportproxy.ExposeTypes) {
	svcKey := ServiceKey(svc)
	svcLog := sb.log.With("service", svcKey)
	// If service has no ready pods associated, don't bother creating any
	// configuration.
	if !hasReadyEndpoints(epSlices) {
		svcLog.Debug("skipping service: it has no running pods")
		return
	}
	// If no ExposeType is given, don't bother creating any configuration.
	if len(expTypes) == 0 {
		svcLog.Debug("skipping service: no expose types provided")
	}

	// Exclude all ports by default, to avoid creating unused clusters.
	var includePorts sets.Set[string]
	// Create listeners for NodePortType
	if expTypes.Has(nodeportproxy.NodePortType) {
		// We only manage NodePort services so Kubernetes takes care of allocating a unique port
		if svc.Spec.Type != corev1.ServiceTypeNodePort {
			svcLog.Warn("skipping service: it is not of type NodePort", "service")
		} else {
			// Add listeners for nodeport services
			ls, ports := sb.makeListenersForNodePortService(svc)
			includePorts = ports.Union(includePorts)
			sb.listeners = append(sb.listeners, ls...)
		}
	}
	// Create filter chains for SNIType
	if expTypes.Has(nodeportproxy.SNIType) && sb.IsSNIEnabled() {
		fcs, ports := sb.makeSNIFilterChains(svcLog, svc)
		includePorts = ports.Union(includePorts)
		sb.fcs = append(sb.fcs, fcs...)
	}
	// Create virtual hosts for TunnelingType
	if expTypes.Has(nodeportproxy.TunnelingType) && sb.IsTunnelingEnabled() {
		vhs, ports := sb.makeTunnelingVirtualHosts(svc)
		includePorts = ports.Union(includePorts)
		sb.vhs = append(sb.vhs, vhs...)
	}

	// Create clusters
	sb.log.Debugw("creating clusters", "includePorts", includePorts)
	sb.clusters = append(sb.clusters, sb.makeClusters(svc, epSlices, includePorts)...)
}

// makeSNIFilterChains returns the FilterChains for the given service and the
// set of ports that are exposed. Note that the set can be nil, don't try to
// write to it before doing a nil check.
func (sb *snapshotBuilder) makeSNIFilterChains(svcLog *zap.SugaredLogger, svc *corev1.Service) ([]*envoylistenerv3.FilterChain, sets.Set[string]) {
	m, err := sb.portHostMappingGetter(svc)
	if err != nil {
		svcLog.Warnw("port host mapping is required with SNI expose type", "error", err)
		return nil, nil
	}
	if err := m.validate(svc); err != nil {
		svcLog.Warnw("port host mapping validation failed", "error", err)
		return nil, nil
	}
	ports, hostnames := m.portHostSets()
	if conflicts := hostnames.Intersection(sets.KeySet(sb.hostnameToService)); len(conflicts) > 0 {
		for _, c := range sets.List(conflicts) {
			svcLog.Warnf("skipping, hostname %q already in use by service %q", c, sb.hostnameToService[c])
			return nil, nil
		}
	}
	// No conflict was detected add the hostname to the map.
	sn := types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}
	for _, h := range sets.List(hostnames) {
		sb.hostnameToService[h] = sn
	}

	svcLog.Debugw("creating sni filter chains", "portHostMapping", m)
	// Besides the filter chains returns the ports that are exposed.
	return makeSNIFilterChains(svc, m), ports
}

// build returns a new Snapshot from the resources derived by the Services
// provided so far.
func (sb *snapshotBuilder) build(version string) (*envoycachev3.Snapshot, error) {
	l, c := sb.makeInitialResources()

	l = append(l, sb.listeners...)
	// Create SNI listener
	if len(sb.fcs) > 0 {
		l = append(l, sb.makeSNIListener(sb.fcs...))
	}
	// Create Tunneling listener
	if len(sb.vhs) > 0 {
		l = append(l, sb.makeTunnelingListener(sb.vhs...))
	}
	c = append(c, sb.clusters...)
	return newSnapshot(version, c, l)
}

func makeAccessLog() []*envoyaccesslogv3.AccessLog {
	f := &envoylistenerlogv3.FileAccessLog{
		Path: "/dev/stdout",
	}

	stdoutAccessLog, err := anypb.New(f)
	if err != nil {
		panic(err)
	}

	accessLog := []*envoyaccesslogv3.AccessLog{
		{
			Name: envoywellknown.FileAccessLog,
			ConfigType: &envoyaccesslogv3.AccessLog_TypedConfig{
				TypedConfig: stdoutAccessLog,
			},
		},
	}
	return accessLog
}

func makeSNIFilterChains(service *corev1.Service, p portHostMapping) []*envoylistenerv3.FilterChain {
	var sniFilterChains []*envoylistenerv3.FilterChain

	serviceKey := ServiceKey(service)
	for _, servicePort := range service.Spec.Ports {
		if name, ok := p[servicePort.Name]; ok {
			servicePortKey := ServicePortKey(serviceKey, &servicePort)

			tcpProxyConfig := &envoytcpfilterv3.TcpProxy{
				StatPrefix: "ingress_tcp",
				ClusterSpecifier: &envoytcpfilterv3.TcpProxy_Cluster{
					Cluster: servicePortKey,
				},
				AccessLog: makeAccessLog(),
			}

			tcpProxyConfigMarshalled, err := anypb.New(tcpProxyConfig)
			if err != nil {
				panic(fmt.Errorf("failed to marshal tcpProxyConfig: %w", err))
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
					ServerNames:       []string{name},
					TransportProtocol: "tls",
				},
			})
		}
	}
	return sniFilterChains
}

func (sb *snapshotBuilder) makeSNIListener(fcs ...*envoylistenerv3.FilterChain) *envoylistenerv3.Listener {
	sb.log.Debugf("using a listener on port %d", sb.EnvoySNIListenerPort)

	inspectorpb, err := anypb.New(&envoytlsinspectorv3.TlsInspector{})
	if err != nil {
		// panic as this either never occurs or cannot recover
		panic(fmt.Errorf("failed to marshal TLS inspector: %w", err))
	}

	sniListener := &envoylistenerv3.Listener{
		Name: "sni_listener",
		Address: &envoycorev3.Address{
			Address: &envoycorev3.Address_SocketAddress{
				SocketAddress: &envoycorev3.SocketAddress{
					Protocol: envoycorev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycorev3.SocketAddress_PortValue{
						PortValue: uint32(sb.EnvoySNIListenerPort),
					},
				},
			},
		},
		// TLS inspector need to be activated explicitly starting from Envoy
		// 1.17.
		ListenerFilters: []*envoylistenerv3.ListenerFilter{
			{
				Name: envoywellknown.TlsInspector,
				ConfigType: &envoylistenerv3.ListenerFilter_TypedConfig{
					TypedConfig: inspectorpb,
				},
			},
		},
		FilterChains: fcs,
	}
	return sniListener
}

func (sb *snapshotBuilder) makeTunnelingVirtualHosts(service *corev1.Service) (vhs []*envoyroutev3.VirtualHost, ports sets.Set[string]) {
	serviceKey := ServiceKey(service)
	ports = sets.New[string]()

	for _, servicePort := range service.Spec.Ports {
		servicePortKey := ServicePortKey(serviceKey, &servicePort)
		if servicePort.Protocol != corev1.ProtocolTCP {
			sb.log.Debugw("skipping servicePort: unsupported protocol", "servicePort", serviceKey, "protocol", corev1.ProtocolTCP)
			continue
		}
		ports.Insert(servicePort.Name)

		vhs = append(vhs, &envoyroutev3.VirtualHost{
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
	return
}

func (sb *snapshotBuilder) makeTunnelingListener(vhs ...*envoyroutev3.VirtualHost) *envoylistenerv3.Listener {
	routerpb, err := anypb.New(&envoyrouterv3.Router{})
	if err != nil {
		// panic as this either never occurs or cannot recover
		panic(fmt.Errorf("failed to marshal router: %w", err))
	}

	hcm := &envoyhttpconnectionmanagerv3.HttpConnectionManager{
		CodecType:  envoyhttpconnectionmanagerv3.HttpConnectionManager_AUTO,
		StatPrefix: "ingress_http",
		RouteSpecifier: &envoyhttpconnectionmanagerv3.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoyroutev3.RouteConfiguration{
				Name:         "http2_connect",
				VirtualHosts: vhs,
			},
		},
		AccessLog: makeAccessLog(),
		HttpFilters: []*envoyhttpconnectionmanagerv3.HttpFilter{
			{
				Name: envoywellknown.Router,
				ConfigType: &envoyhttpconnectionmanagerv3.HttpFilter_TypedConfig{
					TypedConfig: routerpb,
				},
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
	httpManagerConfigMarshalled, err := anypb.New(hcm)
	if err != nil {
		panic(fmt.Errorf("failed to marshal HTTP Connection Manager: %w", err))
	}

	sb.log.Debugf("using a listener on port %d", sb.EnvoyTunnelingListenerPort)

	tunnelingListener := &envoylistenerv3.Listener{
		Name: "tunneling_listener",
		Address: &envoycorev3.Address{
			Address: &envoycorev3.Address_SocketAddress{
				SocketAddress: &envoycorev3.SocketAddress{
					Protocol: envoycorev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycorev3.SocketAddress_PortValue{
						PortValue: uint32(sb.EnvoyTunnelingListenerPort),
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
							TypedConfig: httpManagerConfigMarshalled,
						},
					},
				},
			},
		},
	}
	return tunnelingListener
}

func (sb *snapshotBuilder) makeClusters(service *corev1.Service, epSlices *discoveryv1.EndpointSliceList, includePorts sets.Set[string]) (clusters []envoycachetype.Resource) {
	serviceKey := ServiceKey(service)
	for _, servicePort := range service.Spec.Ports {
		if !includePorts.Has(servicePort.Name) {
			sb.log.Debugw("excluding service port", "servicePort", servicePort.Name)
			continue
		}
		servicePortKey := ServicePortKey(serviceKey, &servicePort)
		endpoints := sb.getEndpointsFromSlices(service, &servicePort, corev1.ProtocolTCP, epSlices)

		// Must be sorted, otherwise we get into trouble when doing the snapshot diff later
		sort.Slice(endpoints, func(i, j int) bool {
			addrI := endpoints[i].HostIdentifier.(*envoyendpointv3.LbEndpoint_Endpoint).Endpoint.Address.Address.(*envoycorev3.Address_SocketAddress).SocketAddress.Address
			addrJ := endpoints[j].HostIdentifier.(*envoyendpointv3.LbEndpoint_Endpoint).Endpoint.Address.Address.(*envoycorev3.Address_SocketAddress).SocketAddress.Address
			return addrI < addrJ
		})

		cluster := &envoyclusterv3.Cluster{
			Name:           servicePortKey,
			ConnectTimeout: durationpb.New(clusterConnectTimeout),
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

func (sb *snapshotBuilder) makeListenersForNodePortService(service *corev1.Service) (listeners []envoycachetype.Resource, exposedPorts sets.Set[string]) {
	serviceKey := ServiceKey(service)
	exposedPorts = sets.New[string]()
	for _, servicePort := range service.Spec.Ports {
		if servicePort.Protocol != corev1.ProtocolTCP {
			sb.log.Debugw("skipping servicePort: unsupported protocol", "servicePort", serviceKey, "protocol", corev1.ProtocolTCP)
			continue
		}
		exposedPorts.Insert(servicePort.Name)
		servicePortKey := ServicePortKey(serviceKey, &servicePort)

		tcpProxyConfig := &envoytcpfilterv3.TcpProxy{
			StatPrefix: "ingress_tcp",
			ClusterSpecifier: &envoytcpfilterv3.TcpProxy_Cluster{
				Cluster: servicePortKey,
			},
		}

		tcpProxyConfigMarshalled, err := anypb.New(tcpProxyConfig)
		if err != nil {
			panic(fmt.Errorf("failed to marshal tcpProxyConfig: %w", err))
		}

		sb.log.Debugw("creating NodePort listener", "service", serviceKey, "nodePort", servicePort.NodePort)

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

func (sb *snapshotBuilder) makeInitialResources() (listeners []envoycachetype.Resource, clusters []envoycachetype.Resource) {
	adminCluster := &envoyclusterv3.Cluster{
		Name:           "service_stats",
		ConnectTimeout: durationpb.New(50 * time.Millisecond),
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
													PortValue: uint32(sb.EnvoyAdminPort),
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
		PassThroughMode: wrapperspb.Bool(false),
		Headers: []*envoyroutev3.HeaderMatcher{
			{
				Name: ":path",
				HeaderMatchSpecifier: &envoyroutev3.HeaderMatcher_StringMatch{
					StringMatch: &envoymatcherv3.StringMatcher{
						MatchPattern: &envoymatcherv3.StringMatcher_Exact{
							Exact: "/healthz",
						},
					},
				},
			},
		},
	}

	healthCheckMarshalled, err := anypb.New(healthCheck)
	if err != nil {
		// panic as this either never occurs or cannot recover
		panic(fmt.Errorf("failed to marshal HealthCheck: %w", err))
	}

	routerpb, err := anypb.New(&envoyrouterv3.Router{})
	if err != nil {
		// panic as this either never occurs or cannot recover
		panic(fmt.Errorf("failed to marshal router: %w", err))
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
				ConfigType: &envoyhttpconnectionmanagerv3.HttpFilter_TypedConfig{
					TypedConfig: routerpb,
				},
			},
		},
	}

	httpConnectionManagerMarshalled, err := anypb.New(httpConnectionManager)
	if err != nil {
		panic(fmt.Errorf("failed to marshal HTTPConnectionManager: %w", err))
	}

	listener := &envoylistenerv3.Listener{
		Name: "service_stats",
		Address: &envoycorev3.Address{
			Address: &envoycorev3.Address_SocketAddress{
				SocketAddress: &envoycorev3.SocketAddress{
					Protocol: envoycorev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycorev3.SocketAddress_PortValue{
						PortValue: uint32(sb.EnvoyStatsPort),
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

	return
}

// hasReadyEndpoints returns true if any EndpointSlice contains at least one ready endpoint.
func hasReadyEndpoints(epSlices *discoveryv1.EndpointSliceList) bool {
	if epSlices == nil {
		return false
	}
	for _, slice := range epSlices.Items {
		for _, endpoint := range slice.Endpoints {
			if endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready {
				return true
			}
		}
	}
	return false
}

// getEndpointsFromSlices returns a slice of LbEndpoint pointers for a given
// service/target port combination from EndpointSlices.
func (sb *snapshotBuilder) getEndpointsFromSlices(s *corev1.Service, port *corev1.ServicePort, proto corev1.Protocol, epSlices *discoveryv1.EndpointSliceList) []*envoyendpointv3.LbEndpoint {
	var upsServers []*envoyendpointv3.LbEndpoint

	if s == nil || port == nil || epSlices == nil {
		return upsServers
	}

	// using a map avoids duplicated upstream servers when the service
	// contains multiple port definitions sharing the same targetport
	processedUpstreamServers := make(map[string]struct{})

	svcKey := ServiceKey(s)
	serviceLog := sb.log.With("service", svcKey)

	for _, slice := range epSlices.Items {
		for _, epPort := range slice.Ports {
			if epPort.Protocol == nil || *epPort.Protocol != proto {
				continue
			}

			var targetPort int32

			// port.Name is optional if there is only one port
			if port.Name == "" || (epPort.Name != nil && port.Name == *epPort.Name) {
				if epPort.Port != nil {
					targetPort = *epPort.Port
				}
			}

			if targetPort <= 0 {
				continue
			}

			for _, endpoint := range slice.Endpoints {
				// Only include ready endpoints
				if endpoint.Conditions.Ready == nil || !*endpoint.Conditions.Ready {
					continue
				}

				for _, address := range endpoint.Addresses {
					ep := net.JoinHostPort(address, strconv.Itoa(int(targetPort)))
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
											Address:  address,
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
	}

	serviceLog.Debugw("endpoints found", "lb-endpoints", upsServers)
	return upsServers
}

func newSnapshot(version string, clusters, listeners []envoycachetype.Resource) (*envoycachev3.Snapshot, error) {
	return envoycachev3.NewSnapshot(
		version,
		map[envoyresourcev3.Type][]envoycachetype.Resource{
			envoyresourcev3.ClusterType:  clusters,
			envoyresourcev3.ListenerType: listeners,
		},
	)
}
