package collectors

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1listers "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

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
}

// MustRegisterClusterCollector registers the cluster collector at the given prometheus registry
func MustRegisterClusterCollector(registry prometheus.Registerer, _ kubeinformers.SharedInformerFactory, kubermaticInformerfactory kubermaticinformers.SharedInformerFactory) {
	cc := &ClusterCollector{
		clusterLister: kubermaticInformerfactory.Kubermatic().V1().Clusters().Lister(),

		clusterCreated: prometheus.NewDesc(
			prefix+"created",
			"Unix creation timestamp",
			[]string{"cluster"}, nil,
		),
		clusterDeleted: prometheus.NewDesc(
			prefix+"deleted",
			"Unix deletion timestamp",
			[]string{"cluster"}, nil,
		),
	}

	registry.MustRegister(cc)
}

// Describe returns the metrics descriptors
func (cc ClusterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cc.clusterCreated
	ch <- cc.clusterDeleted
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
}
