package main

import (
	"context"
	"flag"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"

	envoyv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoydiscoveryv2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	debug               bool
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
	klog.InitFlags(nil)
	flag.BoolVar(&debug, "debug", false, "Use debug logging")
	flag.StringVar(&listenAddress, "listen-address", ":8001", "Address to serve on")
	flag.StringVar(&envoyNodeName, "envoy-node-name", "kube", "Name of the envoy nodes to apply the config to via xds")
	flag.IntVar(&envoyAdminPort, "envoy-admin-port", 9001, "Envoys admin port")
	flag.IntVar(&envoyStatsPort, "envoy-stats-port", 8002, "Limited port which should be opened on envoy to expose metrics and the health check. Endpoints are: /healthz & /stats")
	flag.StringVar(&namespace, "namespace", "", "The namespace we should use for pods and services. Leave empty for all namespaces.")
	flag.StringVar(&exposeAnnotationKey, "expose-annotation-key", defaultExposeAnnotationKey, "The annotation key used to determine if a service should be exposed")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	stopCh := signals.SetupSignalHandler()
	go func() {
		<-stopCh
		cancel()
	}()

	mainLog := logrus.New()
	mainLog.SetLevel(logrus.InfoLevel)
	if debug {
		mainLog.SetLevel(logrus.DebugLevel)
		mainLog.ReportCaller = true
	}

	envoyLog := &envoyLogProxy{
		upstream: mainLog.WithField("component", "envoycache"),
	}

	mainLog.Infof("Starting the server on '%s'...", listenAddress)
	snapshotCache := envoycache.NewSnapshotCache(true, hasher{}, envoyLog)
	srv := xds.NewServer(snapshotCache, nil)
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		mainLog.Fatalf("failed to listen on address '%s': %v", listenAddress, err)
	}

	envoydiscoveryv2.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)
	envoyv2.RegisterEndpointDiscoveryServiceServer(grpcServer, srv)
	envoyv2.RegisterClusterDiscoveryServiceServer(grpcServer, srv)
	envoyv2.RegisterRouteDiscoveryServiceServer(grpcServer, srv)
	envoyv2.RegisterListenerDiscoveryServiceServer(grpcServer, srv)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			mainLog.Fatal(err)
		}
	}()

	config, err := ctrlruntimeconfig.GetConfig()
	if err != nil {
		mainLog.Fatal(err)
	}

	mgr, err := manager.New(config, manager.Options{Namespace: namespace})
	if err != nil {
		mainLog.Fatal(err)
	}

	r := &reconciler{
		ctx:                 ctx,
		Client:              mgr.GetClient(),
		namespace:           namespace,
		envoySnapshotCache:  snapshotCache,
		log:                 mainLog.WithField("annotation", exposeAnnotationKey),
		lastAppliedSnapshot: envoycache.NewSnapshot("v0.0.0", nil, nil, nil, nil),
	}
	ctrl, err := controller.New("envoy-manager", mgr,
		controller.Options{Reconciler: r, MaxConcurrentReconciles: 1})
	if err != nil {
		mainLog.Fatalf("failed to construct mgr: %v", err)
	}

	for _, t := range []runtime.Object{&corev1.Pod{}, &corev1.Service{}} {
		if err := ctrl.Watch(&source.Kind{Type: t}, controllerutil.EnqueueConst("")); err != nil {
			mainLog.Fatalf("failed to watch %t: %v", t, err)
		}
	}

	if err := mgr.Start(stopCh); err != nil {
		mainLog.Printf("Manager ended with err: %v", err)
	}
}
