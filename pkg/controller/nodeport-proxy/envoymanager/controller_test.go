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

	"github.com/go-test/deep"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyroutev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
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

	"k8c.io/kubermatic/v2/pkg/test"
)

func TestSync(t *testing.T) {
	// Used for SNI conflict test
	timeRef := time.Date(2020, time.December, 0, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name                  string
		resources             []runtime.Object
		sniListenerPort       int
		tunnelingListenerPort int
		expectedClusters      map[string]*envoyclusterv3.Cluster
		expectedListener      map[string]*envoylistenerv3.Listener
	}{
		{
			name: "2-ports-2-pods-named-and-non-named-ports",
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					WithServiceType(corev1.ServiceTypeNodePort).
					WithAnnotation(DefaultExposeAnnotationKey, "true").
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					WithAnnotation(DefaultExposeAnnotationKey, "NodePort").
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
			resources: []runtime.Object{
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-nodeport", Namespace: "test"}).
					WithAnnotation(DefaultExposeAnnotationKey, "NodePort").
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-cluster-ip", Namespace: "test"}).
					WithAnnotation(DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(PortHostMappingAnnotationKey, `{"https": "host.com"}`).
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-cluster-ip", Namespace: "test"}).
					WithAnnotation(DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(PortHostMappingAnnotationKey, `{"https": "host.com", "admin": "admin.host.com"}`).
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-cluster-ip", Namespace: "test"}).
					WithAnnotation(DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(PortHostMappingAnnotationKey, `{"https": "host.com"}`).
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "newer-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(PortHostMappingAnnotationKey, `{"https": "host.com"}`).
					WithServicePort("https", 443, 0, intstr.FromString("https"), corev1.ProtocolTCP).
					Build(),
				test.NewEndpointsBuilder(test.NamespacedName{Name: "newer-service", Namespace: "test"}).
					WithEndpointsSubset().
					WithEndpointPort("https", 8443, corev1.ProtocolTCP).
					WithReadyAddressIP("172.16.0.1").
					DoneWithEndpointSubset().Build(),
				test.NewServiceBuilder(test.NamespacedName{Name: "older-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef).
					WithAnnotation(DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(PortHostMappingAnnotationKey, `{"https": "host.com"}`).
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "udp-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(DefaultExposeAnnotationKey, "SNI").
					WithAnnotation(PortHostMappingAnnotationKey, `{"": "host.com"}`).
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(DefaultExposeAnnotationKey, "Tunneling").
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(DefaultExposeAnnotationKey, "SNI,Tunneling").
					WithAnnotation(PortHostMappingAnnotationKey, `{"https": "host.com"}`).
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
			resources: []runtime.Object{
				test.NewServiceBuilder(test.NamespacedName{Name: "my-service", Namespace: "test"}).
					WithCreationTimestamp(timeRef.Add(1*time.Hour)).
					WithAnnotation(DefaultExposeAnnotationKey, "SNI,Tunneling").
					WithAnnotation(PortHostMappingAnnotationKey, `{"http": "host.com"}`). // port http does not exist
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
			client := fakectrlruntimeclient.NewFakeClient(test.resources...)
			c, _, _ := NewReconciler(
				context.TODO(),
				log,
				client,
				Options{
					EnvoyNodeName:              "node-name",
					ExposeAnnotationKey:        DefaultExposeAnnotationKey,
					EnvoySNIListenerPort:       test.sniListenerPort,
					EnvoyTunnelingListenerPort: test.tunnelingListenerPort,
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

		tcpProxyConfigMarshalled, err := ptypes.MarshalAny(tcpProxyConfig)
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
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
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
