package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/nodecsrapprover"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster"

	"github.com/golang/glog"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

type controllerRunOptions struct {
	internalAddr string
	openshift    bool
}

func main() {
	runOp := controllerRunOptions{}
	flag.StringVar(&runOp.internalAddr, "internal-address", "127.0.0.1:8085", "The address on which the internal HTTP /metrics server is running on")
	flag.BoolVar(&runOp.openshift, "openshift", false, "Whether the managed cluster is an openshift cluster")
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
