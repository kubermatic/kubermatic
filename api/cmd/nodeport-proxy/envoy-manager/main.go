package main

import (
	"flag"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	envoyv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoydiscoveryv2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubecache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var (
	debug         bool
	listenAddress string
	envoyNodeName string

	envoyStatsPort int
	envoyAdminPort int
)

const (
	exposeAnnotationKey   = "nodeport-proxy.k8s.io/expose"
	eventValue            = ""
	clusterConnectTimeout = 1 * time.Second
)

func main() {
	flag.BoolVar(&debug, "debug", false, "Use debug logging")
	flag.StringVar(&listenAddress, "listen-address", ":8001", "Address to serve on")
	flag.StringVar(&envoyNodeName, "envoy-node-name", "kube", "Name of the envoy nodes to apply the config to via xds")
	flag.IntVar(&envoyAdminPort, "envoy-admin-port", 9001, "Envoys admin port")
	flag.IntVar(&envoyStatsPort, "envoy-stats-port", 8002, "Limited port which should be opened on envoy to expose metrics and the health check. Endpoints are: /healthz & /stats")
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	mainLog := logrus.New()
	mainLog.SetLevel(logrus.InfoLevel)
	if debug {
		mainLog.SetLevel(logrus.DebugLevel)
		mainLog.ReportCaller = true
	}

	mainLog.Infof("Starting the server on '%s'...", listenAddress)
	snapshotCache := envoycache.NewSnapshotCache(true, hasher{}, mainLog)
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

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		mainLog.Fatal(err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(client, 10*time.Hour)

	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()
	podLister := kubeInformerFactory.Core().V1().Pods().Lister()
	serviceInformer := kubeInformerFactory.Core().V1().Services().Informer()
	serviceLister := kubeInformerFactory.Core().V1().Services().Lister()

	m := controller{
		podLister:           podLister,
		serviceLister:       serviceLister,
		envoySnapshotCache:  snapshotCache,
		syncLock:            &sync.Mutex{},
		log:                 mainLog.WithField("annotation", exposeAnnotationKey),
		lastAppliedSnapshot: envoycache.NewSnapshot("v0.0.0", nil, nil, nil, nil),
		queue:               workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, 5*time.Minute)),
	}

	podInformer.AddEventHandler(kubecache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { m.queue.Add(eventValue) },
		DeleteFunc: func(_ interface{}) { m.queue.Add(eventValue) },
		UpdateFunc: func(_, _ interface{}) { m.queue.Add(eventValue) },
	})
	serviceInformer.AddEventHandler(kubecache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { m.queue.Add(eventValue) },
		DeleteFunc: func(_ interface{}) { m.queue.Add(eventValue) },
		UpdateFunc: func(_, _ interface{}) { m.queue.Add(eventValue) },
	})

	kubeInformerFactory.Start(stopCh)
	kubeInformerFactory.WaitForCacheSync(stopCh)

	m.Run(stopCh)
}
