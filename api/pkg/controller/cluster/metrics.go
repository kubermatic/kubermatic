package cluster

import (
	"sync"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	clusterControllerSubsystem = "kubermatic_cluster_controller"
)

var (
	workers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: clusterControllerSubsystem,
			Name:      "workers",
			Help:      "The number of running cluster controller workers.",
		},
	)

	updates = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: clusterControllerSubsystem,
			Name:      "updates",
			Help:      "The number of times a seed cluster resource was updated.",
		},
		[]string{"cluster", "type", "resource_name"},
	)
)

var registerMetrics sync.Once

// Register the metrics that are to be monitored.
func init() {
	registerMetrics.Do(func() {
		prometheus.MustRegister(workers, updates)
		workers.Set(0)
	})
}

func countSeedResourceUpdate(cluster *kubermaticv1.Cluster, typeName, resourceName string) {
	updates.With(prometheus.Labels{
		"cluster":       cluster.Name,
		"type":          typeName,
		"resource_name": resourceName,
	}).Inc()
}
