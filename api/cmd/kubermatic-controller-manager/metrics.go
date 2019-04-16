package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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

// We have our own metrics server implementation that gathers the metrics from
// both the default prometheus registry and the ctrltuntimemetrics registry.
// The background is that the latter is not configurable at all and we don't
// want to force developers into using it, because that is counterintuitive
// and prone to be forgotten
type metricsServer struct {
	gatherers     prometheus.Gatherers
	listenAddress string
}

func (m *metricsServer) Start(stop <-chan struct{}) error {
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

	go func() {
		<-stop
		if err := s.Shutdown(context.Background()); err != nil {
			glog.Errorf("failed to shutdown metrics server: %v", err)
		}
	}()

	return fmt.Errorf("metrics server stopped: %v", s.ListenAndServe())
}
