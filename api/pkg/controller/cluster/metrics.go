package cluster

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	clusterControllerSubsystem = "kubermatic_cluster_controller"
)

var (
	registerMetrics sync.Once
	workers         = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: clusterControllerSubsystem,
			Name:      "workers",
			Help:      "The number of running cluster controller workers.",
		},
	)
	staleLBs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: clusterControllerSubsystem,
			Name:      "stale_lbs",
			Help:      "The number of cloud load balancers that couldn't be cleaned up within the 2h grace period",
		},
		[]string{"cluster"},
	)
)

// Register the metrics that are to be monitored.
func init() {
	registerMetrics.Do(func() {
		prometheus.MustRegister(workers)
		workers.Set(0)
		prometheus.MustRegister(staleLBs)
	})
}
