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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterPrefix = "kubermatic_cluster_"
)

// ClusterCollector exports metrics for cluster resources.
type ClusterCollector struct {
	client ctrlruntimeclient.Reader

	clusterCreated *prometheus.Desc
	clusterDeleted *prometheus.Desc
	clusterInfo    *prometheus.Desc
	clusterOwner   *prometheus.Desc
}

func newClusterCollector(client ctrlruntimeclient.Reader) *ClusterCollector {
	return &ClusterCollector{
		client: client,
		clusterCreated: prometheus.NewDesc(
			clusterPrefix+"created",
			"Unix creation timestamp",
			[]string{"cluster"},
			nil,
		),
		clusterDeleted: prometheus.NewDesc(
			clusterPrefix+"deleted",
			"Unix deletion timestamp",
			[]string{"cluster"},
			nil,
		),
		clusterInfo: prometheus.NewDesc(
			clusterPrefix+"info",
			"Additional cluster information",
			[]string{
				"name",
				"display_name",
				"ip",
				"spec_version",
				"current_version",
				"cloud_provider",
				"datacenter",
				"pause",
				"project",
				"phase",
			},
			nil,
		),
		clusterOwner: prometheus.NewDesc(
			clusterPrefix+"owner",
			"Synthetic metric that maps clusters to their owners",
			[]string{
				"cluster_name",
				"user",
			},
			nil,
		),
	}
}

// MustRegisterClusterCollector registers the cluster collector at the given prometheus registry.
func MustRegisterClusterCollector(registry prometheus.Registerer, client ctrlruntimeclient.Reader) {
	registry.MustRegister(newClusterCollector(client))
}

// Describe returns the metrics descriptors.
func (cc ClusterCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(cc, ch)
}

// Collect gets called by prometheus to collect the metrics.
func (cc ClusterCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	clusters := &kubermaticv1.ClusterList{}
	if err := cc.client.List(ctx, clusters); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list clusters in ClusterCollector: %w", err))
		return
	}

	users := &kubermaticv1.UserList{}
	if err := cc.client.List(ctx, users); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list users in ClusterCollector: %w", err))
		return
	}

	userMap := map[string]string{}
	for _, user := range users.Items {
		userMap[user.Spec.Email] = user.Name
	}

	kubernetesLabelSet := sets.New[string]()
	for _, cluster := range clusters.Items {
		kubernetesLabelSet = kubernetesLabelSet.Union(sets.KeySet(cluster.Labels))
	}

	kubernetesLabels := caseInsensitiveSort(sets.List(kubernetesLabelSet))

	prometheusLabels := convertToPrometheusLabels(kubernetesLabels)
	labelsGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: clusterPrefix + "labels",
		Help: "Kubernetes labels on Cluster resources",
	}, append([]string{"name"}, prometheusLabels...))

	for _, cluster := range clusters.Items {
		cc.collectCluster(ch, &cluster, kubernetesLabels, userMap, labelsGauge)
	}
}

func (cc *ClusterCollector) collectCluster(ch chan<- prometheus.Metric, c *kubermaticv1.Cluster, kubernetesLabels []string, users map[string]string, labelsGaugeVec *prometheus.GaugeVec) {
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

	var clusterOwner string
	if email := c.Status.UserEmail; email != "" {
		if userName := users[email]; userName != "" {
			clusterOwner = userName
		} else {
			// For regular users the lookup above should never fail, so including the
			// e-mail here should not leak PII, but be helpful for automations with
			// non-existent users.
			clusterOwner = email
		}
	}

	ch <- prometheus.MustNewConstMetric(
		cc.clusterOwner,
		prometheus.GaugeValue,
		1,
		c.Name,
		clusterOwner,
	)

	infoLabels, err := cc.clusterInfoLabels(c)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to determine labels for cluster %s: %w", c.Name, err))
	} else {
		ch <- prometheus.MustNewConstMetric(
			cc.clusterInfo,
			prometheus.GaugeValue,
			1,
			infoLabels...,
		)
	}

	// assemble the labels for this cluster, in the order given by kubernetesLabels, but
	// taking special care of label key conflicts
	clusterLabels := []string{c.Name}
	usedLabels := sets.New[string]()
	for _, key := range kubernetesLabels {
		prometheusLabel := convertToPrometheusLabel(key)
		if !usedLabels.Has(prometheusLabel) {
			clusterLabels = append(clusterLabels, c.Labels[key])
			usedLabels.Insert(prometheusLabel)
		}
	}

	gauge := labelsGaugeVec.WithLabelValues(clusterLabels...)
	gauge.Set(1)
	gauge.Collect(ch)
}

func (cc *ClusterCollector) clusterInfoLabels(cluster *kubermaticv1.Cluster) ([]string, error) {
	provider, err := kubermaticv1helper.ClusterCloudProviderName(cluster.Spec.Cloud)
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
		cluster.Status.Address.IP,
		cluster.Spec.Version.String(),
		cluster.Status.Versions.ControlPlane.String(),
		provider,
		cluster.Spec.Cloud.DatacenterName,
		pause,
		cluster.Labels[kubermaticv1.ProjectIDLabelKey],
		string(cluster.Status.Phase),
	}, nil
}
