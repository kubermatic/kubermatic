package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// We have to unregister the ProcessCollector and GoCollector
// from the ctrltuntimemetrics Registry, otherwise Collecting errors
// out because they are both there and in the default prometheus registry.
// This is not extremely nice but as pretty as "collect metrics from the
// two registries" will ever got, unless the ctrltuntimemetrics.Registry
// becomes configurable
func init() {
	ctrlruntimemetrics.Registry.Unregister(
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	ctrlruntimemetrics.Registry.Unregister(prometheus.NewGoCollector())
}

// New returns a brand new *MetricsServer that gathers the metrics
// from both the prometheus default registry and the ctrlruntimemetrics registry
func New(listenAddress string) *MetricsServer {
	return &MetricsServer{
		gatherers:     []prometheus.Gatherer{prometheus.DefaultGatherer, ctrlruntimemetrics.Registry},
		listenAddress: listenAddress,
	}
}

// MetricsServer is our own metrics server implementation that gathers the metrics from
// both the default prometheus registry and the ctrltuntimemetrics registry.
// The background is that the latter is not configurable at all and we don't
// want to force developers into using it, because that is counterintuitive
// and prone to be forgotten
type MetricsServer struct {
	gatherers     prometheus.Gatherers
	listenAddress string
}

// Start implements sigs.k8s.io/controller-runtime/pkg/manager.Runnable
func (m *MetricsServer) Start(stop <-chan struct{}) error {
	if len(m.gatherers) < 1 {
		return errors.New("no gatherers defined")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer,
		promhttp.HandlerFor(m.gatherers, promhttp.HandlerOpts{}),
	))
	s := http.Server{
		Addr:         m.listenAddress,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return fmt.Errorf("metrics server stopped: %v", s.ListenAndServe())
}

// MetricsServer implements LeaderElectionRunnable to indicate that it does not require to run
// within an elected leader
var _ manager.LeaderElectionRunnable = &MetricsServer{}

func (m *MetricsServer) NeedLeaderElection() bool {
	return false
}
