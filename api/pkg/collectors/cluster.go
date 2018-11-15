package collectors

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1listers "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
)

const (
	prefix = "kubermatic_cluster_"
)

// ClusterCollector exports metrics for cluster resources
type ClusterCollector struct {
	clusterLister kubermaticv1listers.ClusterLister

	clusterCreated *prometheus.Desc
	clusterDeleted *prometheus.Desc
	clusterInfo    *prometheus.Desc
}

// MustRegisterClusterCollector registers the cluster collector at the given prometheus registry
func MustRegisterClusterCollector(registry prometheus.Registerer, _ kubeinformers.SharedInformerFactory, kubermaticInformerfactory kubermaticinformers.SharedInformerFactory) {
	cc := &ClusterCollector{
		clusterLister: kubermaticInformerfactory.Kubermatic().V1().Clusters().Lister(),
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
				"user_name",
				"user_email",
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
	cacheClusters, err := cc.clusterLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list clusters from clusterLister in ClusterCollector: %v", err))
		return
	}

	for _, cacheCluster := range cacheClusters {
		cluster := cacheCluster.DeepCopy()

		cc.collectCluster(ch, cluster)
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

	return []string{
		cluster.Name,
		cluster.Spec.HumanReadableName,
		cluster.Address.IP,
		cluster.Spec.Version.String(),
		provider,
		cluster.Spec.Cloud.DatacenterName,
		cluster.Status.UserName,
		cluster.Status.UserEmail,
	}, nil
}
