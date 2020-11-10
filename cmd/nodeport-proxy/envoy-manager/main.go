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
	"flag"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	envoycachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	xdsv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"

	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/cli"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	namespace           string
	listenAddress       string
	envoyNodeName       string
	exposeAnnotationKey string

	envoyStatsPort int
	envoyAdminPort int
)

const (
	defaultExposeAnnotationKey = "nodeport-proxy.k8s.io/expose"
	clusterConnectTimeout      = 1 * time.Second
)

func main() {
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	flag.StringVar(&listenAddress, "listen-address", ":8001", "Address to serve on")
	flag.StringVar(&envoyNodeName, "envoy-node-name", "kube", "Name of the envoy nodes to apply the config to via xds")
	flag.IntVar(&envoyAdminPort, "envoy-admin-port", 9001, "Envoys admin port")
	flag.IntVar(&envoyStatsPort, "envoy-stats-port", 8002, "Limited port which should be opened on envoy to expose metrics and the health check. Endpoints are: /healthz & /stats")
	flag.StringVar(&namespace, "namespace", "", "The namespace we should use for pods and services. Leave empty for all namespaces.")
	flag.StringVar(&exposeAnnotationKey, "expose-annotation-key", defaultExposeAnnotationKey, "The annotation key used to determine if a service should be exposed")
	flag.Parse()

	// setup signal handler
	ctx, cancel := context.WithCancel(context.Background())
	stopCh := signals.SetupSignalHandler()
	go func() {
		<-stopCh
		cancel()
	}()

	// init logging
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	cli.Hello(log, "Envoy-Manager", logOpts.Debug)
	log.Infow("Starting the server...", "address", listenAddress)

	snapshotCache := envoycachev3.NewSnapshotCache(true, hasher{}, log.With("component", "envoycache"))
	srv := xdsv3.NewServer(ctx, snapshotCache, nil)
	grpcServer := grpc.NewServer()

	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalw("failed to listen on address", zap.Error(err))
	}

	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, srv)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, srv)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, srv)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, srv)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	config, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	mgr, err := manager.New(config, manager.Options{Namespace: namespace})
	if err != nil {
		log.Fatal(err)
	}

	r := &reconciler{
		ctx:                ctx,
		Client:             mgr.GetClient(),
		namespace:          namespace,
		envoySnapshotCache: snapshotCache,
		log:                log,
	}
	ctrl, err := controller.New("envoy-manager", mgr,
		controller.Options{Reconciler: r, MaxConcurrentReconciles: 1})
	if err != nil {
		log.Fatalw("failed to construct mgr", zap.Error(err))
	}

	for _, t := range []runtime.Object{&corev1.Pod{}, &corev1.Service{}} {
		if err := ctrl.Watch(&source.Kind{Type: t}, controllerutil.EnqueueConst("")); err != nil {
			log.Fatalw("failed to watch", "kind", t, zap.Error(err))
		}
	}

	if err := mgr.Start(stopCh); err != nil {
		log.Errorw("Manager ended with err", zap.Error(err))
	}
}
