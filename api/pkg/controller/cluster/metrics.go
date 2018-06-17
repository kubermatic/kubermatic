package cluster

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricNamespace = "kubermatic"
)

// Metrics contains metrics about the clusters & workers
type Metrics struct {
	Clusters        prometheus.Gauge
	ClusterPhases   *prometheus.GaugeVec
	Workers         prometheus.Gauge
	UnhandledErrors prometheus.Counter
}

// NewMetrics creates a new ControllerMetrics
// with default values initialized, so metrics always show up.
func NewMetrics(registerMetrics bool) *Metrics {
	subsystem := "cluster_controller"

	cm := &Metrics{
		Clusters: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "clusters",
			Help:      "The number of currently managed clusters",
		}),
		ClusterPhases: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "cluster_status_phase",
			Help:      "All phases a cluster can be in. 1 if the cluster is in that phase",
		}, []string{"cluster", "phase"}),
		Workers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running cluster controller workers",
		}),
		UnhandledErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "unhandled_errors_total",
			Help:      "The number of unhandled errors that occurred in the controller's reconciliation loop",
		}),
	}

	// Set default values, so that these metrics always show up
	cm.Clusters.Set(0)
	cm.Workers.Set(0)

	if registerMetrics {
		prometheus.MustRegister(cm.Clusters)
		prometheus.MustRegister(cm.ClusterPhases)
		prometheus.MustRegister(cm.Workers)
		prometheus.MustRegister(cm.UnhandledErrors)
	}

	return cm
}
