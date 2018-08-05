package cluster

import (
	"fmt"
	"sync"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/util/runtime"
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

	clusters = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: clusterControllerSubsystem,
			Name:      "cluster_info",
			Help:      "The clusters managed by this controller.",
		},
		[]string{"name", "display_name", "ip", "master_version", "cloud_provider", "user_name", "user_email"},
	)
)

var (
	registerMetrics    sync.Once
	clusterInfoPerName map[string]prometheus.Labels
)

// Register the metrics that are to be monitored.
func init() {
	clusterInfoPerName = make(map[string]prometheus.Labels)

	registerMetrics.Do(func() {
		prometheus.MustRegister(workers, updates, clusters)
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

func metricsLabelsForCluster(cluster *kubermaticv1.Cluster) (prometheus.Labels, error) {
	provider, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
	if err != nil {
		return prometheus.Labels{}, err
	}

	return prometheus.Labels{
		"name":           cluster.Name,
		"display_name":   cluster.Spec.HumanReadableName,
		"ip":             cluster.Address.IP,
		"master_version": cluster.Spec.Version,
		"cloud_provider": provider,
		"user_name":      cluster.Status.UserName,
		"user_email":     cluster.Status.UserEmail,
	}, nil
}

// deleteClusterMetrics attempts to find a previous metric for a given
// cluster and remove it from the metrics vector. This is important for
// cases where cluster labels change over time, like the IP address or
// version. In order to not leave old metrics laying around, we need to
// ensure that there is always only one metric per cluster name.
func deleteClusterMetrics(labels prometheus.Labels, update bool) {
	name := labels["name"]

	oldMetric, exists := clusterInfoPerName[name]
	if exists {
		clusters.Delete(oldMetric)
	}

	if update {
		clusterInfoPerName[name] = labels
	} else {
		delete(clusterInfoPerName, name)
	}
}

// countClusterInMetrics adds a metrics for a cluster, taking care
// of removing any previously existing metric for the same cluster.
func countClusterInMetrics(cluster *kubermaticv1.Cluster) {
	labels, err := metricsLabelsForCluster(cluster)
	if err != nil {
		runtime.HandleError(fmt.Errorf("could not determine cluster cloud provider: %v", err))
		return
	}

	deleteClusterMetrics(labels, true)
	clusters.With(labels).Set(1)
}

// removeClusterFromMetrics removes the cluster metric after a
// cluster has been removed.
func removeClusterFromMetrics(cluster *kubermaticv1.Cluster) {
	labels, err := metricsLabelsForCluster(cluster)
	if err != nil {
		runtime.HandleError(fmt.Errorf("could not determine cluster cloud provider: %v", err))
		return
	}

	deleteClusterMetrics(labels, false)
}
