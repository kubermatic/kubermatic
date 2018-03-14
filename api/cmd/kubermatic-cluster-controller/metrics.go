package main

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

// ClusterControllerMetrics is a struct of all metrics used in
// the cluster controller.
type ClusterControllerMetrics struct {
	Clusters        metrics.Gauge
	ClusterPhases   metrics.Gauge
	Workers         metrics.Gauge
	UnhandledErrors metrics.Counter
}

// NewClusterControllerMetrics creates new ClusterControllerMetrics
// with default values initialized, so metrics always show up.
func NewClusterControllerMetrics() *ClusterControllerMetrics {
	namespace := "kubermatic"
	subsystem := "cluster_controller"

	cm := &ClusterControllerMetrics{
		Clusters: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "clusters",
			Help:      "The number of currently managed clusters",
		}, nil),
		ClusterPhases: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cluster_status_phase",
			Help:      "All phases a cluster can be in. 1 if the cluster is in that phase",
		}, []string{"cluster", "phase"}),
		Workers: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running cluster controller workers",
		}, []string{}),
		UnhandledErrors: prometheus.NewCounterFrom(prom.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "unhandled_errors",
			Help:      "The number of unhandled errors that occurred in the controller's reconciliation loop",
		}, []string{}),
	}

	// Set default values, so that these metrics always show up
	cm.Clusters.Set(0)
	cm.Workers.Set(0)

	return cm
}
