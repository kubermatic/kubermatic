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
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyroutev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoyhealthv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	envoyhttpconnectionmanagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	envoytcpfilterv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	envoyresourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	//Need to change to cache/v2 for v2 but also no .Resource, nor in v2
	// nor in v3

	envoycachetype "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	envoywellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type reconciler struct {
	ctrlruntimeclient.Client

	ctx       context.Context
	log       *zap.SugaredLogger
	namespace string

	envoySnapshotCache  envoycachev3.SnapshotCache
	lastAppliedSnapshot envoycachev3.Snapshot
}

func (r *reconciler) getInitialResources() (listeners []envoycachetype.Resource, clusters []envoycachetype.Resource, err error) {
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
		return nil, nil, errors.Wrap(err, "failed to marshal HealthCheck")
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
		return nil, nil, errors.Wrap(err, "failed to marshal HTTPConnectionManager")
	}

	listener := &envoylistenerv3.Listener{
		Name: "service_stats",
		Address: &envoycorev3.Address{
			Address: &envoycorev3.Address_SocketAddress{
				SocketAddress: &envoycorev3.SocketAddress{
					Protocol: envoycorev3.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycorev3.SocketAddress_PortValue{
						PortValue: uint32(envoyStatsPort),
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

	return listeners, clusters, nil
}

func (r *reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	r.log.Debug("Got reconcile request")
	err := r.sync()
	if err != nil {
		r.log.Errorf("Failed to reconcile", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (r *reconciler) sync() error {
	services := &corev1.ServiceList{}
	if err := r.List(r.ctx, services, ctrlruntimeclient.InNamespace(r.namespace)); err != nil {
		return errors.Wrap(err, "failed to list service's")
	}

	listeners, clusters, err := r.getInitialResources()
	if err != nil {
		return errors.Wrap(err, "failed to get initial config")
	}

	for _, service := range services.Items {
		serviceKey := ServiceKey(&service)
		serviceLog := r.log.With("service", serviceKey)

		// Only cover services which have the annotation: true
		if strings.ToLower(service.Annotations[exposeAnnotationKey]) != "true" {
			serviceLog.Debugf("Skipping service: it does not have the annotation %s=true", exposeAnnotationKey)
			continue
		}

		// We only manage NodePort services so Kubernetes takes care of allocating a unique port
		if service.Spec.Type != corev1.ServiceTypeNodePort {
			serviceLog.Warn("Skipping service: it is not of type NodePort")
			return nil
		}

		pods, err := r.getReadyServicePods(&service)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get pod's for service '%s'", serviceKey))
		}

		// If we have no pods, dont bother creating a cluster.
		if len(pods) == 0 {
			serviceLog.Debug("Skipping service: it has no running pods")
			continue
		}

		for _, servicePort := range service.Spec.Ports {
			serviceNodePortName := fmt.Sprintf("%s-%d", serviceKey, servicePort.NodePort)
			servicePortLog := serviceLog.With("port", servicePort.NodePort)

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
				return errors.Wrap(err, "failed to marshal tcpProxyConfig")
			}

			r.log.Debugf("Using a listener on port %d", servicePort.NodePort)

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
	}

	lastUsedVersion, err := semver.NewVersion(r.lastAppliedSnapshot.GetVersion(envoyresourcev3.ClusterType))
	if err != nil {
		return errors.Wrap(err, "failed to parse version from last snapshot")
	}

	// Generate a new snapshot using the old version to be able to do a DeepEqual comparison
	//TODO(youssefazrak) add the needed arguments
	snapshot := envoycachev3.NewSnapshot(lastUsedVersion.String(), nil, clusters, nil, listeners, nil, nil)
	if equality.Semantic.DeepEqual(r.lastAppliedSnapshot, snapshot) {
		return nil
	}

	r.log.Info("detected a change. Updating the Envoy config cache...")
	newVersion := lastUsedVersion.IncMajor()
	newSnapshot := envoycachev3.NewSnapshot(newVersion.String(), nil, clusters, nil, listeners, nil, nil)

	if err := newSnapshot.Consistent(); err != nil {
		return errors.Wrap(err, "new Envoy config snapshot is not consistent")
	}

	if err := r.envoySnapshotCache.SetSnapshot(envoyNodeName, newSnapshot); err != nil {
		return errors.Wrap(err, "failed to set a new Envoy cache snapshot")
	}

	r.lastAppliedSnapshot = newSnapshot

	return nil
}

func (r *reconciler) getReadyServicePods(service *corev1.Service) ([]*corev1.Pod, error) {
	key := ServiceKey(service)
	var readyPods []*corev1.Pod

	// As we take the service selector as input, we can assume that its validated
	opts := &ctrlruntimeclient.ListOptions{
		LabelSelector: labels.SelectorFromValidatedSet(service.Spec.Selector),
		Namespace:     service.Namespace,
	}
	servicePods := &corev1.PodList{}
	if err := r.List(context.Background(), servicePods, opts); err != nil {
		return readyPods, errors.Wrap(err, fmt.Sprintf("failed to list pod's for service '%s' via selector: '%s'", key, opts.LabelSelector.String()))
	}

	if len(servicePods.Items) == 0 {
		return readyPods, nil
	}

	// Filter for ready pods
	for idx, pod := range servicePods.Items {
		if PodIsReady(&pod) {
			readyPods = append(readyPods, &servicePods.Items[idx])
		} else {
			// Only log when we do not add pods as the caller is responsible for logging the final pods
			r.log.Debugf("Skipping pod %s/%s for service %s/%s. The pod is not ready", pod.Namespace, pod.Name, service.Namespace, service.Name)
		}
	}

	return readyPods, nil
}
