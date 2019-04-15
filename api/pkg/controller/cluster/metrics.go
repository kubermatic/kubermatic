package cluster

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	ctrlruntimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
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
)

// Register the metrics that are to be monitored.
func init() {
	registerMetrics.Do(func() {
		ctrlruntimemetrics.Registry.MustRegister(workers)
		workers.Set(0)
		ctrlruntimemetrics.Registry.MustRegister(staleLBs)
	})
}
