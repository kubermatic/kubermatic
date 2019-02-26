package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	envoyv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoycorev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoyendpointv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoylistenerv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoyroutev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoyhealthv2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/health_check/v2"
	envoyhttpconnectionmanagerv2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoytcpfilterv2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	envoyutil "github.com/envoyproxy/go-control-plane/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	log   *logrus.Entry
	queue workqueue.RateLimitingInterface

	podLister     corev1lister.PodLister
	serviceLister corev1lister.ServiceLister

	syncLock            *sync.Mutex
	envoySnapshotCache  envoycache.SnapshotCache
	lastAppliedSnapshot envoycache.Snapshot
}

func (c *controller) Run(stopCh <-chan struct{}) {
	defer c.queue.ShutDown()

	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	defer c.queue.Done(key)

	if err := c.Sync(); err != nil {
		c.log.Errorf("failed to sync: %v", err)
		c.queue.AddRateLimited(key)
	}

	return true
}

func (c *controller) getInitialResources() (listeners []envoycache.Resource, clusters []envoycache.Resource, err error) {
	adminCluster := &envoyv2.Cluster{
		Name:           "service_stats",
		ConnectTimeout: 50 * time.Millisecond,
		Type:           envoyv2.Cluster_STATIC,
		LbPolicy:       envoyv2.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoyv2.ClusterLoadAssignment{
			ClusterName: "service_stats",
			Endpoints: []envoyendpointv2.LocalityLbEndpoints{
				{
					LbEndpoints: []envoyendpointv2.LbEndpoint{
						{
							HostIdentifier: &envoyendpointv2.LbEndpoint_Endpoint{
								Endpoint: &envoyendpointv2.Endpoint{
									Address: &envoycorev2.Address{
										Address: &envoycorev2.Address_SocketAddress{
											SocketAddress: &envoycorev2.SocketAddress{
												Protocol: envoycorev2.TCP,
												Address:  "127.0.0.1",
												PortSpecifier: &envoycorev2.SocketAddress_PortValue{
													PortValue: uint32(envoyAdminPort),
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

	healthCheck := &envoyhealthv2.HealthCheck{
		PassThroughMode: &types.BoolValue{Value: false},
		Headers: []*envoyroutev2.HeaderMatcher{
			{
				Name: ":path",
				HeaderMatchSpecifier: &envoyroutev2.HeaderMatcher_ExactMatch{
					ExactMatch: "/healthz",
				},
			},
		},
	}
	healthCheckMsg, err := envoyutil.MessageToStruct(healthCheck)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to convert HealthCheck config to GRPC message")
	}

	httpConnectionManager := &envoyhttpconnectionmanagerv2.HttpConnectionManager{
		CodecType:  envoyhttpconnectionmanagerv2.AUTO,
		StatPrefix: "service_stats",
		RouteSpecifier: &envoyhttpconnectionmanagerv2.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoyv2.RouteConfiguration{
				VirtualHosts: []envoyroutev2.VirtualHost{
					{
						Name:    "backend",
						Domains: []string{"*"},
						Routes: []envoyroutev2.Route{
							{
								Match: envoyroutev2.RouteMatch{
									PathSpecifier: &envoyroutev2.RouteMatch_Prefix{
										Prefix: "/stats",
									},
								},
								Action: &envoyroutev2.Route_Route{
									Route: &envoyroutev2.RouteAction{
										ClusterSpecifier: &envoyroutev2.RouteAction_Cluster{
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
		HttpFilters: []*envoyhttpconnectionmanagerv2.HttpFilter{
			{
				Name:       envoyutil.HealthCheck,
				ConfigType: &envoyhttpconnectionmanagerv2.HttpFilter_Config{Config: healthCheckMsg},
			},
			{
				Name: envoyutil.Router,
			},
		},
	}

	httpConnectionManagerMsg, err := envoyutil.MessageToStruct(httpConnectionManager)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to convert HTTPConnectionManager config to GRPC message")
	}

	listener := &envoyv2.Listener{
		Name: "service_stats",
		Address: envoycorev2.Address{
			Address: &envoycorev2.Address_SocketAddress{
				SocketAddress: &envoycorev2.SocketAddress{
					Protocol: envoycorev2.TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycorev2.SocketAddress_PortValue{
						PortValue: uint32(envoyStatsPort),
					},
				},
			},
		},
		FilterChains: []envoylistenerv2.FilterChain{
			{
				Filters: []envoylistenerv2.Filter{
					{
						Name: envoyutil.HTTPConnectionManager,
						ConfigType: &envoylistenerv2.Filter_Config{
							Config: httpConnectionManagerMsg,
						},
					},
				},
			},
		},
	}
	listeners = append(listeners, listener)

	return listeners, clusters, nil
}

func (c *controller) Sync() error {
	c.syncLock.Lock()
	defer c.syncLock.Unlock()

	listerServices, err := c.serviceLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "failed to list service's from lister")
	}

	listeners, clusters, err := c.getInitialResources()
	if err != nil {
		return errors.Wrap(err, "failed to get initial config")
	}

	for _, service := range listerServices {
		serviceKey := ServiceKey(service)

		// Only cover services which have the annotation: true
		if strings.ToLower(service.Annotations[exposeAnnotationKey]) != "true" {
			c.log.Debugf("Skipping service '%s'. It does not have the annotation %s=true", serviceKey, exposeAnnotationKey)
			continue
		}

		// We only manage NodePort services so Kubernetes takes care of allocating a unique port
		if service.Spec.Type != corev1.ServiceTypeNodePort {
			c.log.Warnf("Skipping service '%s'. It is not of type NodePort", serviceKey)
			return nil
		}

		pods, err := c.getReadyServicePods(service)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get pod's for service '%s'", serviceKey))
		}

		// If we have no pods, dont bother creating a cluster.
		if len(pods) == 0 {
			c.log.Debugf("skipping service %s/%s as it has no running pods", service.Namespace, service.Name)
			continue
		}

		for _, servicePort := range service.Spec.Ports {
			serviceNodePortName := fmt.Sprintf("%s-%d", serviceKey, servicePort.NodePort)

			var endpoints []envoyendpointv2.LbEndpoint
			for _, pod := range pods {

				// Get the port on the pod, the NodePort Service port is pointing to
				podPort := getMatchingPodPort(servicePort, pod)
				if podPort == 0 {
					c.log.Infof("Skipping pod %s/%s for service port %s/%s:%d. The service port does not match to any of the pods containers", pod.Namespace, pod.Name, service.Namespace, service.Name, servicePort.NodePort)
					continue
				}

				c.log.Debugf("Using pod %s/%s:%d as backend for %s/%s:%d", pod.Namespace, pod.Name, podPort, service.Namespace, service.Name, servicePort.NodePort)

				// Cluster endpoints
				endpoints = append(endpoints, envoyendpointv2.LbEndpoint{
					HostIdentifier: &envoyendpointv2.LbEndpoint_Endpoint{
						Endpoint: &envoyendpointv2.Endpoint{
							Address: &envoycorev2.Address{
								Address: &envoycorev2.Address_SocketAddress{
									SocketAddress: &envoycorev2.SocketAddress{
										Protocol: envoycorev2.TCP,
										Address:  pod.Status.PodIP,
										PortSpecifier: &envoycorev2.SocketAddress_PortValue{
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
				addrI := endpoints[i].HostIdentifier.(*envoyendpointv2.LbEndpoint_Endpoint).Endpoint.Address.Address.(*envoycorev2.Address_SocketAddress).SocketAddress.Address
				addrJ := endpoints[j].HostIdentifier.(*envoyendpointv2.LbEndpoint_Endpoint).Endpoint.Address.Address.(*envoycorev2.Address_SocketAddress).SocketAddress.Address
				return addrI < addrJ
			})

			cluster := &envoyv2.Cluster{
				Name:           serviceNodePortName,
				ConnectTimeout: clusterConnectTimeout,
				Type:           envoyv2.Cluster_STATIC,
				LbPolicy:       envoyv2.Cluster_ROUND_ROBIN,
				LoadAssignment: &envoyv2.ClusterLoadAssignment{
					ClusterName: serviceNodePortName,
					Endpoints: []envoyendpointv2.LocalityLbEndpoints{
						{
							LbEndpoints: endpoints,
						},
					},
				},
			}
			clusters = append(clusters, cluster)

			tcpProxyConfig := &envoytcpfilterv2.TcpProxy{
				StatPrefix: "ingress_tcp",
				ClusterSpecifier: &envoytcpfilterv2.TcpProxy_Cluster{
					Cluster: serviceNodePortName,
				},
			}
			tcpProxyConfigStruct, err := envoyutil.MessageToStruct(tcpProxyConfig)
			if err != nil {
				return errors.Wrap(err, "failed to convert TCPProxy config to GRPC struct")
			}

			c.log.Debugf("Using a listener on port %d", servicePort.NodePort)

			listener := &envoyv2.Listener{
				Name: serviceNodePortName,
				Address: envoycorev2.Address{
					Address: &envoycorev2.Address_SocketAddress{
						SocketAddress: &envoycorev2.SocketAddress{
							Protocol: envoycorev2.TCP,
							Address:  "0.0.0.0",
							PortSpecifier: &envoycorev2.SocketAddress_PortValue{
								PortValue: uint32(servicePort.NodePort),
							},
						},
					},
				},
				FilterChains: []envoylistenerv2.FilterChain{
					{
						Filters: []envoylistenerv2.Filter{
							{
								Name: envoyutil.TCPProxy,
								ConfigType: &envoylistenerv2.Filter_Config{
									Config: tcpProxyConfigStruct,
								},
							},
						},
					},
				},
			}
			listeners = append(listeners, listener)
		}
	}

	lastUsedVersion, err := semver.NewVersion(c.lastAppliedSnapshot.GetVersion(envoycache.ClusterType))
	if err != nil {
		return errors.Wrap(err, "failed to parse version from last snapshot")
	}

	// Generate a new snapshot using the old version to be able to do a DeepEqual comparison
	snapshot := envoycache.NewSnapshot(lastUsedVersion.String(), nil, clusters, nil, listeners)
	if equality.Semantic.DeepEqual(c.lastAppliedSnapshot, snapshot) {
		return nil
	}

	c.log.Info("detected a change. Updating the Envoy config cache...")
	newVersion := lastUsedVersion.IncMajor()
	newSnapshot := envoycache.NewSnapshot(newVersion.String(), nil, clusters, nil, listeners)

	if err := newSnapshot.Consistent(); err != nil {
		return errors.Wrap(err, "new Envoy config snapshot is not consistent")
	}

	err = c.envoySnapshotCache.SetSnapshot(envoyNodeName, newSnapshot)
	if err != nil {
		return errors.Wrap(err, "failed to set a new Envoy cache snapshot")
	}

	c.lastAppliedSnapshot = newSnapshot

	return nil
}

func (c *controller) getReadyServicePods(service *corev1.Service) ([]*corev1.Pod, error) {
	key := ServiceKey(service)
	var readyPods []*corev1.Pod

	// As we take the service selector as input, we can assume that its validated
	selector := labels.SelectorFromValidatedSet(service.Spec.Selector)
	servicePods, err := c.podLister.Pods(service.Namespace).List(selector)
	if err != nil {
		return readyPods, errors.Wrap(err, fmt.Sprintf("failed to list pod's for service '%s' via selector: '%s'", key, selector.String()))
	}

	if len(servicePods) == 0 {
		return readyPods, nil
	}

	// Filter for ready pods
	for _, pod := range servicePods {
		if PodIsReady(pod) {
			readyPods = append(readyPods, pod)
		} else {
			// Only log when we do not add pods as the caller is responsible for logging the final pods
			c.log.Debugf("Skipping pod %s/%s for service %s/%s. The pod ist not ready", pod.Namespace, pod.Name, service.Namespace, service.Name)
		}
	}

	return readyPods, nil
}
