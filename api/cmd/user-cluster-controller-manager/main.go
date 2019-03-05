package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/ipam"
	"github.com/kubermatic/kubermatic/api/pkg/controller/nodecsrapprover"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster"

	"github.com/golang/glog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

type controllerRunOptions struct {
	internalAddr string
	openshift    bool
	networks     networkFlags
}

func main() {
	runOp := controllerRunOptions{}
	flag.StringVar(&runOp.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal HTTP /metrics server is running on")
	flag.BoolVar(&runOp.openshift, "openshift", false, "Whether the managed cluster is an openshift cluster")
	flag.Var(&runOp.networks, "ipam-controller-network", `The networks from which the ipam controller should allocate IPs for machines (e.g.: .--ipam-controller-network="10.0.0.0/16,10.0.0.1,8.8.8.8" --ipam-controller-network="192.168.5.0/24,192.168.5.1,1.1.1.1,8.8.4.4")`)
	flag.Parse()

	var g run.Group

	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatal(err)
	}
	stopCh := signals.SetupSignalHandler()
	ctx, ctxDone := context.WithCancel(context.Background())
	defer ctxDone()

	// Create Context
	done := ctx.Done()

	mgr, err := manager.New(cfg, manager.Options{LeaderElection: true, LeaderElectionNamespace: metav1.NamespaceSystem})
	if err != nil {
		glog.Fatal(err)
	}

	glog.Info("registering components")
	if err := apiextensionv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}
	if err := apiregistrationv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatal(err)
	}

	metrics := newMetrics()

	// Setup all Controllers
	glog.Info("registering controllers")
	if _, err := usercluster.Add(mgr, runOp.openshift); err != nil {
		glog.Fatalf("failed to register user cluster controller: %v", err)
	}
	metrics.controllers.Inc()
	if len(runOp.networks) > 0 {
		if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			glog.Fatalf("failed to add clusterv1alpha1 scheme: %v", err)
		}
		if err := ipam.Add(mgr, runOp.networks); err != nil {
			glog.Fatalf("failed to add IPAM controller to mgr: %v", err)
		}
		glog.Infof("Added IPAM controller to mgr")
		metrics.controllers.Inc()
	}
	if err := registerControllers(mgr, metrics); err != nil {
		glog.Fatal(err)
	}
	if runOp.openshift {
		if err := nodecsrapprover.Add(mgr, 4, cfg); err != nil {
			glog.Fatalf("failed to add nodecsrapprover controller: %v", err)
		}
		glog.Infof("Registered nodecsrapprover controller")
	}

	// This group is forever waiting in a goroutine for signals to stop
	{
		g.Add(func() error {
			select {
			case <-stopCh:
				return errors.New("user requested to stop the application")
			case <-done:
				return errors.New("parent context has been closed - propagating the request")
			}
		}, func(err error) {
			ctxDone()
		})
	}

	// This group is running an internal http server with metrics and other debug information
	{
		m := http.NewServeMux()
		m.Handle("/metrics", promhttp.Handler())

		s := http.Server{
			Addr:         runOp.internalAddr,
			Handler:      m,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		g.Add(func() error {
			glog.Infof("starting the internal HTTP metrics server: %s\n", runOp.internalAddr)
			return s.ListenAndServe()
		}, func(err error) {
			glog.Infof("stopping internal HTTP metrics server, err = %v", err)
			timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := s.Shutdown(timeoutCtx); err != nil {
				glog.Errorf("failed to shutdown the internal HTTP server gracefully, err = %v", err)
			}
		})
	}

	// This group starts the controller manager
	{
		g.Add(func() error {
			// Start the Cmd
			return mgr.Start(done)
		}, func(err error) {
			glog.Infof("stopping user cluster controller manager, err = %v", err)
		})
	}

	if err := g.Run(); err != nil {
		glog.Fatal(err)
	}

}
