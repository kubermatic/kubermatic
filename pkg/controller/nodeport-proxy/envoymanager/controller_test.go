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
	"time"

	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyroutev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoyhttpconnectionmanagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	envoytcpfilterv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoyresourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	envoywellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestSync(t *testing.T) {
	// Used for SNI conflict test
	timeRef := time.Date(2020, time.December, 0, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name                  string
		resources             []ctrlruntimeclient.Object
		sniListenerPort       int
		tunnelingListenerPort int
		expectedClusters      map[string]*envoyclusterv3.Cluster
		expectedListener      map[string]*envoylistenerv3.Listener
	}{
		{
			name: "2-ports-2-pods-named-and-non-named-ports",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					WithServiceType(corev1.ServiceTypeNodePort).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "true").
					WithServicePort("https", 443, 32000, intstr.FromString("https"), corev1.ProtocolTCP).
					WithServicePort("http", 80, 32001, intstr.FromString("http"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithEndpointPort("http", 8080, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					WithReadyAddressIP("172.16.0.2").
					DoneWithEndpointSubset().Build(),
			},
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-nodeport-https": makeCluster(t, "test/my-nodeport-https", 8443, "172.16.0.1", "172.16.0.2"),
				"test/my-nodeport-http":  makeCluster(t, "test/my-nodeport-http", 8080, "172.16.0.1", "172.16.0.2"),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"test/my-nodeport-https": makeNodePortListener(t, "test/my-nodeport-https", 32000),
				"test/my-nodeport-http":  makeNodePortListener(t, "test/my-nodeport-http", 32001),
			},
		},
		{
			name: "1-port-2-pods-one-unhealthy",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "NodePort").
					WithServiceType(corev1.ServiceTypeNodePort).
					WithServicePort("http", 80, 32001, intstr.FromString("http"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("http", 8080, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					WithNotReadyAddressIP("172.16.0.2").
					DoneWithEndpointSubset().Build(),
			},
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-nodeport-http": makeCluster(t, "test/my-nodeport-http", 8080, "172.16.0.1"),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"test/my-nodeport-http": makeNodePortListener(t, "test/my-nodeport-http", 32001),
			},
		},
		{
			name: "1-port-service-without-annotation",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					WithServiceType(corev1.ServiceTypeNodePort).
					WithServicePort("http", 80, 32001, intstr.FromString("http"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					Build(),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{},
			expectedClusters: map[string]*envoyclusterv3.Cluster{},
		},
		{
			name: "1-port-service-without-pods",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "NodePort").
					WithServiceType(corev1.ServiceTypeNodePort).
					WithServicePort("http", 80, 32001, intstr.FromString("http"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					Build(),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{},
			expectedClusters: map[string]*envoyclusterv3.Cluster{},
		},
		{
			name: "1-sni-service-with-1-exposed-port",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-cluster-ip", Namespace: "test"}).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"https": "host.com"}`).
					WithServicePort("http", 80, 0, intstr.FromString("http"), corev1.ProtocolTCP).
					WithServicePort("https", 8080, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-cluster-ip", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("http", 8080, corev1.ProtocolTCP).
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					WithReadyAddressIP("172.16.0.2").
					DoneWithEndpointSubset().Build(),
			},
			sniListenerPort: 443,
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-cluster-ip-https": makeCluster(t, "test/my-cluster-ip-https", 8443, "172.16.0.1", "172.16.0.2"),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"sni_listener": makeSNIListener(t, 443, hostClusterName{Cluster: "test/my-cluster-ip-https", Hostname: "host.com"}),
			},
		},
		{
			name: "1-sni-service-with-2-exposed-ports",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-cluster-ip", Namespace: "test"}).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"https": "host.com", "admin": "admin.host.com"}`).
					WithServicePort("http", 80, 0, intstr.FromString("http"), corev1.ProtocolTCP).
					WithServicePort("https", 8080, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					WithServicePort("admin", 6443, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-cluster-ip", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("http", 8080, corev1.ProtocolTCP).
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithEndpointPort("admin", 6443, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					WithReadyAddressIP("172.16.0.2").
					DoneWithEndpointSubset().Build(),
			},
			sniListenerPort: 443,
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-cluster-ip-https": makeCluster(t, "test/my-cluster-ip-https", 8443, "172.16.0.1", "172.16.0.2"),
				"test/my-cluster-ip-admin": makeCluster(t, "test/my-cluster-ip-admin", 6443, "172.16.0.1", "172.16.0.2"),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"sni_listener": makeSNIListener(t, 443,
					hostClusterName{Cluster: "test/my-cluster-ip-https", Hostname: "host.com"},
					hostClusterName{Cluster: "test/my-cluster-ip-admin", Hostname: "admin.host.com"}),
			},
		},
		{
			name: "sni-listener-not-enabled",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-cluster-ip", Namespace: "test"}).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"https": "host.com"}`).
					WithServicePort("https", 8080, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-cluster-ip", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					DoneWithEndpointSubset().Build(),
			},
			// 0 deactivates the SNI listener
			sniListenerPort:  0,
			expectedClusters: map[string]*envoyclusterv3.Cluster{},
			expectedListener: map[string]*envoylistenerv3.Listener{},
		},
		{
			name: "sni-hostname-conflict",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "newer-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"https": "host.com"}`).
					WithServicePort("https", 443, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "newer-service", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					DoneWithEndpointSubset().Build(),
				test.NewServiceBuilder(test.NamespacedName{Name: "older-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"https": "host.com"}`).
					WithServicePort("https", 443, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "older-service", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.2").
					DoneWithEndpointSubset().Build(),
			},
			sniListenerPort: 443,
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/older-service-https": makeCluster(t, "test/older-service-https", 8443, "172.16.0.2"),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"sni_listener": makeSNIListener(t, 443,
					hostClusterName{Cluster: "test/older-service-https", Hostname: "host.com"}),
			},
		},
		{
			name: "sni-udp-port",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "udp-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"": "host.com"}`).
					WithServicePort("", 1025, 0, intstr.FromString(""), corev1.ProtocolUDP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "udp-service", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("https", 1025, corev1.ProtocolUDP).
					WithReadyAddressIP("172.16.0.1").
					DoneWithEndpointSubset().Build(),
			},
			sniListenerPort:  443,
			expectedClusters: map[string]*envoyclusterv3.Cluster{},
			expectedListener: map[string]*envoylistenerv3.Listener{},
		},
		{
			name: "tunneling",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "Tunneling").
					WithServicePort("https", 443, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-service", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					DoneWithEndpointSubset().Build(),
			},
			tunnelingListenerPort: 443,
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-service-https": makeCluster(t, "test/my-service-https", 8443, "172.16.0.1"),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"tunneling_listener": makeTunnelingListener(t, 443, hostClusterName{Cluster: "test/my-service-https", Hostname: "my-service.test.svc.cluster.local:443"}),
			},
		},
		{
			name: "both-sni-and-tunneling",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "SNI,Tunneling").
					WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"https": "host.com"}`).
					WithServicePort("https", 443, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-service", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					DoneWithEndpointSubset().Build(),
			},
			tunnelingListenerPort: 8080,
			sniListenerPort:       8443,
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-service-https": makeCluster(t, "test/my-service-https", 8443, "172.16.0.1"),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"tunneling_listener": makeTunnelingListener(t, 8080, hostClusterName{Cluster: "test/my-service-https", Hostname: "my-service.test.svc.cluster.local:443"}),
				"sni_listener":       makeSNIListener(t, 8443, hostClusterName{Cluster: "test/my-service-https", Hostname: "host.com"}),
			},
		},
		{
			name: "both-sni-and-http2-connect-invalid-sni-mapping",
			resources: []ctrlruntimeclient.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(nodeportproxy.DefaultExposeAnnotationKey, "SNI,Tunneling").
					WithAnnotation(nodeportproxy.PortHostMappingAnnotationKey, `{"http": "host.com"}`). // port http does not exist
					WithServicePort("https", 443, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "my-service", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					DoneWithEndpointSubset().Build(),
			},
			tunnelingListenerPort: 8080,
			sniListenerPort:       8443,
			expectedClusters: map[string]*envoyclusterv3.Cluster{
				"test/my-service-https": makeCluster(t, "test/my-service-https", 8443, "172.16.0.1"),
			},
			expectedListener: map[string]*envoylistenerv3.Listener{
				"tunneling_listener": makeTunnelingListener(t, 8080, hostClusterName{Cluster: "test/my-service-https", Hostname: "my-service.test.svc.cluster.local:443"}),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log := zaptest.NewLogger(t).Sugar()
			ctx := context.Background()
			client := fake.
				NewClientBuilder().
				WithObjects(test.resources...).
				WithIndex(&corev1.Service{}, nodeportproxy.DefaultExposeAnnotationKey, func(raw ctrlruntimeclient.Object) []string {
					svc := raw.(*corev1.Service)
					if isExposed(svc, nodeportproxy.DefaultExposeAnnotationKey) {
						return []string{"true"}
					}
					return nil
				}).
				Build()
			c, _, _ := NewReconciler(
				ctx,
				log,
				client,
				Options{
					EnvoyNodeName:              "node-name",
					ExposeAnnotationKey:        nodeportproxy.DefaultExposeAnnotationKey,
					EnvoySNIListenerPort:       test.sniListenerPort,
					EnvoyTunnelingListenerPort: test.tunnelingListenerPort,
				},
			)

			if err := c.sync(ctx); err != nil {
				t.Fatalf("failed to execute controller sync func: %v", err)
			}

			gotClusters := map[string]*envoyclusterv3.Cluster{}
			s, _ := c.cache.GetSnapshot(c.options.EnvoyNodeName)

			for name, res := range s.GetResources(envoyresourcev3.ClusterType) {
				gotClusters[name] = res.(*envoyclusterv3.Cluster)
			}
			// Delete the admin cluster. We're not going to bother comparing it here, as its a static resource.
			// It would just pollute the testing code
			delete(gotClusters, "service_stats")

			if d := diff.ObjectDiff(test.expectedClusters, gotClusters); d != "" {
				t.Errorf("Got unexpected clusters:\n%v", d)
			}

			gotListeners := map[string]*envoylistenerv3.Listener{}
			for name, res := range s.GetResources(envoyresourcev3.ListenerType) {
				gotListeners[name] = res.(*envoylistenerv3.Listener)
			}
			delete(gotListeners, "service_stats")

			if d := diff.ObjectDiff(test.expectedListener, gotListeners); d != "" {
				t.Errorf("Got unexpected listeners:\n%v", d)
			}
		})
	}
}

func makeNodePortListener(t *testing.T, name string, portValue uint32) *envoylistenerv3.Listener {
	return &envoylistenerv3.Listener{
		Name: name,
		Address: &envoycorev3.Address{
			Address: &envoycorev3.Address_SocketAddress{
				SocketAddress: &envoycorev3.SocketAddress{
					Protocol: envoycorev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycorev3.SocketAddress_PortValue{
						PortValue: portValue,
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
									Cluster: name,
								},
							}),
						},
					},
				},
			},
		},
	}
}

type hostClusterName struct {
	Hostname string
	Cluster  string
}

func makeSNIListener(t *testing.T, portValue uint32, hostClusterNames ...hostClusterName) *envoylistenerv3.Listener {
	fcs := []*envoylistenerv3.FilterChain{}
	for _, hc := range hostClusterNames {
		tcpProxyConfig := &envoytcpfilterv3.TcpProxy{
			StatPrefix: "ingress_tcp",
			ClusterSpecifier: &envoytcpfilterv3.TcpProxy_Cluster{
				Cluster: hc.Cluster,
			},
			AccessLog: makeAccessLog(),
		}

		tcpProxyConfigMarshalled, err := anypb.New(tcpProxyConfig)
		if err != nil {
			t.Fatalf("failed to marshal tcpProxyConfig: %v", err)
		}

		fcs = append(fcs, &envoylistenerv3.FilterChain{
			Filters: []*envoylistenerv3.Filter{
				{
					Name: envoywellknown.TCPProxy,
					ConfigType: &envoylistenerv3.Filter_TypedConfig{
						TypedConfig: tcpProxyConfigMarshalled,
					},
				},
			},
			FilterChainMatch: &envoylistenerv3.FilterChainMatch{
				ServerNames:       []string{hc.Hostname},
				TransportProtocol: "tls",
			},
		})
	}
	sb := &snapshotBuilder{}
	sb.EnvoySNIListenerPort = int(portValue)
	sb.log = zaptest.NewLogger(t).Sugar()
	return sb.makeSNIListener(fcs...)
}

func makeTunnelingListener(t *testing.T, portValue int, hostClusterNames ...hostClusterName) *envoylistenerv3.Listener {
	var vhs []*envoyroutev3.VirtualHost
	for _, hostClusterName := range hostClusterNames {
		vhs = append(vhs, &envoyroutev3.VirtualHost{
			Name:    hostClusterName.Cluster,
			Domains: []string{hostClusterName.Hostname},
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
								Cluster: hostClusterName.Cluster,
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
	sb := &snapshotBuilder{}
	sb.EnvoyTunnelingListenerPort = portValue
	sb.log = zaptest.NewLogger(t).Sugar()
	return sb.makeTunnelingListener(vhs...)
}

func makeCluster(t *testing.T, name string, portValue uint32, addresses ...string) *envoyclusterv3.Cluster {
	lbs := []*envoyendpointv3.LbEndpoint{}
	for _, address := range addresses {
		lbs = append(lbs, &envoyendpointv3.LbEndpoint{
			HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
				Endpoint: &envoyendpointv3.Endpoint{
					Address: &envoycorev3.Address{
						Address: &envoycorev3.Address_SocketAddress{
							SocketAddress: &envoycorev3.SocketAddress{
								Protocol: envoycorev3.SocketAddress_TCP,
								Address:  address,
								PortSpecifier: &envoycorev3.SocketAddress_PortValue{
									PortValue: portValue,
								},
							},
						},
					},
				},
			},
		})
	}
	return &envoyclusterv3.Cluster{
		Name:           name,
		ConnectTimeout: durationpb.New(clusterConnectTimeout),
		ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{
			Type: envoyclusterv3.Cluster_STATIC,
		},
		LbPolicy: envoyclusterv3.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
			ClusterName: name,
			Endpoints: []*envoyendpointv3.LocalityLbEndpoints{
				{
					LbEndpoints: lbs,
				},
			},
		},
	}
}

func TestNewEndpointHandler(t *testing.T) {
	tests := []struct {
		name          string
		eps           *corev1.Endpoints
		resources     []ctrlruntimeclient.Object
		expectResults []ctrlruntime.Request
	}{
		{
			name:          "No results when matching service is not found",
			eps:           &corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "foo"}},
			expectResults: nil,
		},
		{
			name: "No result when matching service found but not exposed",
			eps:  &corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "foo"}},
			resources: []ctrlruntimeclient.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "foo"},
				},
			},
			expectResults: nil,
		},
		{
			name: "Result expected when exposed matching service is found",
			eps:  &corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}},
			resources: []ctrlruntimeclient.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   "foo",
						Name:        "bar",
						Annotations: map[string]string{nodeportproxy.DefaultExposeAnnotationKey: "true"},
					},
				},
			},
			expectResults: []ctrlruntime.Request{{
				NamespacedName: types.NamespacedName{Namespace: "foo", Name: "bar"},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := zaptest.NewLogger(t).Sugar()
			client := fake.
				NewClientBuilder().
				WithObjects(tt.resources...).
				Build()

			handler := (&Reconciler{
				options: Options{ExposeAnnotationKey: nodeportproxy.DefaultExposeAnnotationKey},
				client:  client,
				log:     log,
			}).newEndpointHandler()

			res := handler(context.Background(), tt.eps)

			if d := diff.ObjectDiff(tt.expectResults, res); d != "" {
				t.Errorf("Got unexpected results:\n%v", d)
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
					Annotations: map[string]string{nodeportproxy.DefaultExposeAnnotationKey: "true"},
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
					Annotations: map[string]string{nodeportproxy.DefaultExposeAnnotationKey: "false"},
				},
			},
			expectAccept: false,
		},
		{
			name: "Should be rejected when annotation value has wrong case",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{nodeportproxy.DefaultExposeAnnotationKey: "True"},
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
				tt.annotationKey = nodeportproxy.DefaultExposeAnnotationKey
			}
			p := exposeAnnotationPredicate{annotation: tt.annotationKey, log: zaptest.NewLogger(t).Sugar()}
			if got, exp := p.Create(event.CreateEvent{Object: tt.obj}), tt.expectAccept; got != exp {
				t.Errorf("expect create accepted %t, but got %t for object: %+v", exp, got, *tt.obj)
			}
			if got, exp := p.Delete(event.DeleteEvent{Object: tt.obj}), tt.expectAccept; got != exp {
				t.Errorf("expect delete accepted %t, but got %t for object: %+v", exp, got, *tt.obj)
			}
			if got, exp := p.Update(event.UpdateEvent{ObjectNew: tt.obj}), tt.expectAccept; got != exp {
				t.Errorf("expect update accepted %t, but got %t for object: %+v", exp, got, *tt.obj)
			}
			if got, exp := p.Generic(event.GenericEvent{Object: tt.obj}), tt.expectAccept; got != exp {
				t.Errorf("expect generic accepted %t, but got %t for object: %+v", exp, got, *tt.obj)
			}
		})
	}
}

func TestConnectionSettings(t *testing.T) {
	svc := test.NewServiceBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
		WithServiceType(corev1.ServiceTypeNodePort).
		WithServicePort("https", 443, 32000, intstr.FromString("https"), corev1.ProtocolTCP).
		Build()

	eps := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-nodeport",
			Namespace: "test",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{IP: "172.16.0.1"},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "https",
						Port:     8443,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		},
	}

	vh := &envoyroutev3.VirtualHost{
		Name:    "test/my-nodeport-https",
		Domains: []string{"my-nodeport.test.svc.cluster.local:443"},
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
							Cluster: "test/my-nodeport-https",
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
	}

	tests := []struct {
		name    string
		options Options
		assert  func(t *testing.T, sb *snapshotBuilder)
	}{
		{
			name: "no_settings_use_envoy_defaults",
			options: Options{
				EnvoySNIListenerPort:       6443,
				EnvoyTunnelingListenerPort: 8088,
			},
			assert: func(t *testing.T, sb *snapshotBuilder) {
				t.Helper()

				filterChains := makeSNIFilterChains(svc, portHostMapping{"https": "host.com"}, sb.GetSNIListenerIdleTimeout())
				if len(filterChains) != 1 {
					t.Fatalf("expected exactly one filter chain, got %d", len(filterChains))
				}

				tcpProxyAny := filterChains[0].Filters[0].GetTypedConfig()
				tcpProxy := &envoytcpfilterv3.TcpProxy{}
				if err := tcpProxyAny.UnmarshalTo(tcpProxy); err != nil {
					t.Fatalf("failed to unmarshal tcp proxy config: %v", err)
				}
				if tcpProxy.GetIdleTimeout() != nil {
					t.Fatalf("expected SNI tcp proxy idle timeout to be unset when not configured")
				}

				sniListener := sb.makeSNIListener(filterChains...)
				if len(sniListener.GetSocketOptions()) != 0 {
					t.Fatalf("expected SNI listener tcp keepalive to be unset when not configured")
				}

				tunnelingListener := sb.makeTunnelingListener(vh)
				if len(tunnelingListener.GetSocketOptions()) != 0 {
					t.Fatalf("expected tunneling listener tcp keepalive to be unset when not configured")
				}

				hcmAny := tunnelingListener.FilterChains[0].Filters[0].GetTypedConfig()
				hcm := &envoyhttpconnectionmanagerv3.HttpConnectionManager{}
				if err := hcmAny.UnmarshalTo(hcm); err != nil {
					t.Fatalf("failed to unmarshal HTTP connection manager config: %v", err)
				}
				if hcm.GetCommonHttpProtocolOptions() != nil {
					t.Fatalf("expected tunneling connection idle timeout to be unset when not configured")
				}
				if hcm.GetStreamIdleTimeout() != nil {
					t.Fatalf("expected tunneling stream idle timeout to be unset when not configured")
				}

				nodePortListeners, _ := sb.makeListenersForNodePortService(svc)
				if len(nodePortListeners) != 1 {
					t.Fatalf("expected exactly one nodeport listener, got %d", len(nodePortListeners))
				}
				nodePortListener, ok := nodePortListeners[0].(*envoylistenerv3.Listener)
				if !ok {
					t.Fatalf("expected listener resource type %T, got %T", &envoylistenerv3.Listener{}, nodePortListeners[0])
				}
				if len(nodePortListener.GetSocketOptions()) != 0 {
					t.Fatalf("expected nodeport listener tcp keepalive to be unset when not configured")
				}

				clusters := sb.makeClusters(svc, eps, sets.New("https"))
				if len(clusters) != 1 {
					t.Fatalf("expected exactly one cluster, got %d", len(clusters))
				}
				cluster, ok := clusters[0].(*envoyclusterv3.Cluster)
				if !ok {
					t.Fatalf("expected cluster resource type %T, got %T", &envoyclusterv3.Cluster{}, clusters[0])
				}
				if cluster.GetUpstreamConnectionOptions() != nil {
					t.Fatalf("expected cluster upstream tcp keepalive to be unset when not configured")
				}
			},
		},
		{
			name: "explicit_timeouts_and_keepalive_are_applied",
			options: Options{
				EnvoySNIListenerPort:              6443,
				EnvoyTunnelingListenerPort:        8088,
				SNIListenerIdleTimeout:            15 * time.Minute,
				TunnelingConnectionIdleTimeout:    15 * time.Minute,
				TunnelingConnectionStreamTimeout:  5 * time.Minute,
				DownstreamTCPKeepaliveTime:        5 * time.Minute,
				DownstreamTCPKeepaliveInterval:    30 * time.Second,
				DownstreamTCPKeepaliveProbes:      5,
				UpstreamTCPKeepaliveTime:          5 * time.Minute,
				UpstreamTCPKeepaliveProbeInterval: 30 * time.Second,
				UpstreamTCPKeepaliveProbeAttempts: 5,
			},
			assert: func(t *testing.T, sb *snapshotBuilder) {
				t.Helper()

				filterChains := makeSNIFilterChains(svc, portHostMapping{"https": "host.com"}, sb.GetSNIListenerIdleTimeout())
				tcpProxyAny := filterChains[0].Filters[0].GetTypedConfig()
				tcpProxy := &envoytcpfilterv3.TcpProxy{}
				if err := tcpProxyAny.UnmarshalTo(tcpProxy); err != nil {
					t.Fatalf("failed to unmarshal tcp proxy config: %v", err)
				}
				if got, want := tcpProxy.GetIdleTimeout().AsDuration(), 15*time.Minute; got != want {
					t.Fatalf("unexpected SNI tcp proxy idle timeout: got %s, want %s", got, want)
				}

				sniListener := sb.makeSNIListener(filterChains...)
				assertListenerTCPKeepaliveSocketOptions(t, sniListener.GetSocketOptions(), 5, 5*time.Minute, 30*time.Second)

				tunnelingListener := sb.makeTunnelingListener(vh)
				assertListenerTCPKeepaliveSocketOptions(t, tunnelingListener.GetSocketOptions(), 5, 5*time.Minute, 30*time.Second)

				hcmAny := tunnelingListener.FilterChains[0].Filters[0].GetTypedConfig()
				hcm := &envoyhttpconnectionmanagerv3.HttpConnectionManager{}
				if err := hcmAny.UnmarshalTo(hcm); err != nil {
					t.Fatalf("failed to unmarshal HTTP connection manager config: %v", err)
				}
				if got, want := hcm.GetCommonHttpProtocolOptions().GetIdleTimeout().AsDuration(), 15*time.Minute; got != want {
					t.Fatalf("unexpected tunneling connection idle timeout: got %s, want %s", got, want)
				}
				if got, want := hcm.GetStreamIdleTimeout().AsDuration(), 5*time.Minute; got != want {
					t.Fatalf("unexpected tunneling stream idle timeout: got %s, want %s", got, want)
				}

				nodePortListeners, _ := sb.makeListenersForNodePortService(svc)
				nodePortListener := nodePortListeners[0].(*envoylistenerv3.Listener)
				assertListenerTCPKeepaliveSocketOptions(t, nodePortListener.GetSocketOptions(), 5, 5*time.Minute, 30*time.Second)

				clusters := sb.makeClusters(svc, eps, sets.New("https"))
				cluster := clusters[0].(*envoyclusterv3.Cluster)
				assertTCPKeepalive(t, cluster.GetUpstreamConnectionOptions().GetTcpKeepalive(), 5, 5*time.Minute, 30*time.Second)
			},
		},
		{
			name: "partial_keepalive_does_not_backfill_missing_values",
			options: Options{
				EnvoySNIListenerPort:              6443,
				DownstreamTCPKeepaliveInterval:    45 * time.Second,
				UpstreamTCPKeepaliveProbeAttempts: 7,
			},
			assert: func(t *testing.T, sb *snapshotBuilder) {
				t.Helper()

				sniListener := sb.makeSNIListener()
				assertListenerTCPKeepaliveSocketOptions(t, sniListener.GetSocketOptions(), 0, 0, 45*time.Second)

				clusters := sb.makeClusters(svc, eps, sets.New("https"))
				cluster := clusters[0].(*envoyclusterv3.Cluster)
				upstreamKeepalive := cluster.GetUpstreamConnectionOptions().GetTcpKeepalive()
				if upstreamKeepalive == nil {
					t.Fatalf("expected upstream tcp keepalive settings to be configured")
				}
				if got, want := upstreamKeepalive.GetKeepaliveProbes().GetValue(), uint32(7); got != want {
					t.Fatalf("unexpected upstream keepalive probes: got %d, want %d", got, want)
				}
				if upstreamKeepalive.KeepaliveTime != nil {
					t.Fatalf("expected upstream keepalive time to be unset when not configured")
				}
				if upstreamKeepalive.KeepaliveInterval != nil {
					t.Fatalf("expected upstream keepalive interval to be unset when not configured")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sb := &snapshotBuilder{
				Options: tc.options,
				log:     zaptest.NewLogger(t).Sugar(),
			}

			tc.assert(t, sb)
		})
	}
}

func assertListenerTCPKeepaliveSocketOptions(t *testing.T, socketOptions []*envoycorev3.SocketOption, probes uint32, keepaliveTime, keepaliveInterval time.Duration) {
	t.Helper()

	soKeepAlive, ok := findSocketOptionIntValue(socketOptions, socketLevelSOLSocket, socketNameSOKeepAlive)
	if !ok || soKeepAlive != 1 {
		t.Fatalf("expected SO_KEEPALIVE=1, got exists=%t value=%d", ok, soKeepAlive)
	}

	if keepaliveTime > 0 {
		got, ok := findSocketOptionIntValue(socketOptions, socketLevelIPProtoTCP, socketNameTCPKeepIdle)
		if !ok || got != int64(keepaliveTime/time.Second) {
			t.Fatalf("unexpected TCP_KEEPIDLE: exists=%t value=%d", ok, got)
		}
	} else if _, ok := findSocketOptionIntValue(socketOptions, socketLevelIPProtoTCP, socketNameTCPKeepIdle); ok {
		t.Fatalf("expected TCP_KEEPIDLE to be unset")
	}

	if keepaliveInterval > 0 {
		got, ok := findSocketOptionIntValue(socketOptions, socketLevelIPProtoTCP, socketNameTCPKeepIntvl)
		if !ok || got != int64(keepaliveInterval/time.Second) {
			t.Fatalf("unexpected TCP_KEEPINTVL: exists=%t value=%d", ok, got)
		}
	} else if _, ok := findSocketOptionIntValue(socketOptions, socketLevelIPProtoTCP, socketNameTCPKeepIntvl); ok {
		t.Fatalf("expected TCP_KEEPINTVL to be unset")
	}

	if probes > 0 {
		got, ok := findSocketOptionIntValue(socketOptions, socketLevelIPProtoTCP, socketNameTCPKeepCnt)
		if !ok || got != int64(probes) {
			t.Fatalf("unexpected TCP_KEEPCNT: exists=%t value=%d", ok, got)
		}
	} else if _, ok := findSocketOptionIntValue(socketOptions, socketLevelIPProtoTCP, socketNameTCPKeepCnt); ok {
		t.Fatalf("expected TCP_KEEPCNT to be unset")
	}
}

func findSocketOptionIntValue(socketOptions []*envoycorev3.SocketOption, level, name int64) (int64, bool) {
	for _, option := range socketOptions {
		if option.GetLevel() != level || option.GetName() != name {
			continue
		}

		intValue, ok := option.GetValue().(*envoycorev3.SocketOption_IntValue)
		if !ok {
			return 0, false
		}

		return intValue.IntValue, true
	}

	return 0, false
}

func assertTCPKeepalive(t *testing.T, tcpKeepalive *envoycorev3.TcpKeepalive, probes uint32, keepaliveTime, keepaliveInterval time.Duration) {
	t.Helper()

	if tcpKeepalive == nil {
		t.Fatalf("expected tcp keepalive settings to be configured")
	}

	if got, want := tcpKeepalive.GetKeepaliveProbes().GetValue(), probes; got != want {
		t.Fatalf("unexpected keepalive probes: got %d, want %d", got, want)
	}

	if got, want := tcpKeepalive.GetKeepaliveTime().GetValue(), uint32(keepaliveTime/time.Second); got != want {
		t.Fatalf("unexpected keepalive time: got %d, want %d", got, want)
	}

	if got, want := tcpKeepalive.GetKeepaliveInterval().GetValue(), uint32(keepaliveInterval/time.Second); got != want {
		t.Fatalf("unexpected keepalive interval: got %d, want %d", got, want)
	}
}

func marshalMessage(t *testing.T, msg proto.Message) *anypb.Any {
	marshalled, err := anypb.New(msg)
	if err != nil {
		t.Fatalf("failed to marshal from message to any: %v", err)
	}

	return marshalled
}
