package collectors

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	prefix = "kubermatic_cluster_"
)

// ClusterCollector exports metrics for cluster resources
type ClusterCollector struct {
	client ctrlruntimeclient.Reader

	clusterCreated *prometheus.Desc
	clusterDeleted *prometheus.Desc
	clusterInfo    *prometheus.Desc
}

// MustRegisterClusterCollector registers the cluster collector at the given prometheus registry
func MustRegisterClusterCollector(registry prometheus.Registerer, client ctrlruntimeclient.Reader) {
	cc := &ClusterCollector{
		client: client,
		clusterCreated: prometheus.NewDesc(
			prefix+"created",
			"Unix creation timestamp",
			[]string{"cluster"},
			nil,
		),
		clusterDeleted: prometheus.NewDesc(
			prefix+"deleted",
			"Unix deletion timestamp",
			[]string{"cluster"},
			nil,
		),
		clusterInfo: prometheus.NewDesc(
			prefix+"info",
			"Cluster information like owner or version",
			[]string{
				"name",
				"display_name",
				"ip",
				"master_version",
				"cloud_provider",
				"datacenter",
				"pause",
				"type",
			},
			nil,
		),
	}

	registry.MustRegister(cc)
}

// Describe returns the metrics descriptors
func (cc ClusterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cc.clusterCreated
	ch <- cc.clusterDeleted
	ch <- cc.clusterInfo
}

// Collect gets called by prometheus to collect the metrics
func (cc ClusterCollector) Collect(ch chan<- prometheus.Metric) {
	clusters := &kubermaticv1.ClusterList{}
	if err := cc.client.List(context.Background(), clusters); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list clusters from clusterLister in ClusterCollector: %v", err))
		return
	}

	for _, cluster := range clusters.Items {
		cc.collectCluster(ch, &cluster)
	}
}

func (cc *ClusterCollector) collectCluster(ch chan<- prometheus.Metric, c *kubermaticv1.Cluster) {
	ch <- prometheus.MustNewConstMetric(
		cc.clusterCreated,
		prometheus.GaugeValue,
		float64(c.CreationTimestamp.Unix()),
		c.Name,
	)

	if c.DeletionTimestamp != nil {
		ch <- prometheus.MustNewConstMetric(
			cc.clusterDeleted,
			prometheus.GaugeValue,
			float64(c.DeletionTimestamp.Unix()),
			c.Name,
		)
	}

	labels, err := cc.clusterLabels(c)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to determine labels for cluster %s: %v", c.Name, err))
	} else {
		ch <- prometheus.MustNewConstMetric(
			cc.clusterInfo,
			prometheus.GaugeValue,
			1,
			labels...,
		)
	}
}

func (cc *ClusterCollector) clusterLabels(cluster *kubermaticv1.Cluster) ([]string, error) {
	provider, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	pause := "false"
	if cluster.Spec.Pause {
		pause = "true"
	}

	clusterType := "kubernetes"
	if cluster.Spec.Openshift != nil {
		clusterType = "openshift"
	}

	return []string{
		cluster.Name,
		cluster.Spec.HumanReadableName,
		cluster.Address.IP,
		cluster.Spec.Version.String(),
		provider,
		cluster.Spec.Cloud.DatacenterName,
		pause,
		clusterType,
	}, nil
}
