package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	envoyv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoydiscoveryv2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
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

	log.Infow("Starting the server...", "address", listenAddress)

	snapshotCache := envoycache.NewSnapshotCache(true, hasher{}, log.With("component", "envoycache"))
	srv := xds.NewServer(ctx, snapshotCache, nil)
	grpcServer := grpc.NewServer()

	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalw("failed to listen on address", zap.Error(err))
	}

	envoydiscoveryv2.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)
	envoyv2.RegisterEndpointDiscoveryServiceServer(grpcServer, srv)
	envoyv2.RegisterClusterDiscoveryServiceServer(grpcServer, srv)
	envoyv2.RegisterRouteDiscoveryServiceServer(grpcServer, srv)
	envoyv2.RegisterListenerDiscoveryServiceServer(grpcServer, srv)

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
		ctx:                 ctx,
		Client:              mgr.GetClient(),
		namespace:           namespace,
		envoySnapshotCache:  snapshotCache,
		log:                 log,
		lastAppliedSnapshot: envoycache.NewSnapshot("v0.0.0", nil, nil, nil, nil, nil),
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
