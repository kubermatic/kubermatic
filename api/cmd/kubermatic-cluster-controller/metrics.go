package main

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

// ClusterControllerMetrics is a struct of all metrics used in
// the cluster controller.
type ClusterControllerMetrics struct {
	Clusters metrics.Gauge
	Workers  metrics.Gauge
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
		}, []string{}),
		Workers: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running cluster controller workers",
		}, []string{}),
	}

	// Set default values, so that these metrics always show up
	cm.Clusters.Set(0)
	cm.Workers.Set(0)

	return cm
}
