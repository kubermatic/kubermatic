/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collectors

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

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

	return []string{
		cluster.Name,
		cluster.Spec.HumanReadableName,
		cluster.Address.IP,
		cluster.Spec.Version.String(),
		provider,
		cluster.Spec.Cloud.DatacenterName,
		pause,
		"kubernetes",
	}, nil
}
