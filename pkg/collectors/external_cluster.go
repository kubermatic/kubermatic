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

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	externalClusterPrefix = "kubermatic_external_cluster_"
)

// ExternalClusterCollector exports metrics for external cluster resources.
type ExternalClusterCollector struct {
	client ctrlruntimeclient.Reader

	clusterCreated *prometheus.Desc
	clusterDeleted *prometheus.Desc
	clusterInfo    *prometheus.Desc
}

// MustRegisterExternalClusterCollector registers the cluster collector at the given prometheus registry.
func MustRegisterExternalClusterCollector(registry prometheus.Registerer, client ctrlruntimeclient.Reader) {
	cc := &ExternalClusterCollector{
		client: client,
		clusterCreated: prometheus.NewDesc(
			externalClusterPrefix+"created",
			"Unix creation timestamp",
			[]string{"cluster"},
			nil,
		),
		clusterDeleted: prometheus.NewDesc(
			externalClusterPrefix+"deleted",
			"Unix deletion timestamp",
			[]string{"cluster"},
			nil,
		),
		clusterInfo: prometheus.NewDesc(
			externalClusterPrefix+"info",
			"Additional external cluster information",
			[]string{
				"name",
				"display_name",
				"provider",
				"phase",
			},
			nil,
		),
	}

	registry.MustRegister(cc)
}

// Describe returns the metrics descriptors.
func (cc ExternalClusterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cc.clusterCreated
	ch <- cc.clusterDeleted
	ch <- cc.clusterInfo
}

// Collect gets called by prometheus to collect the metrics.
func (cc ExternalClusterCollector) Collect(ch chan<- prometheus.Metric) {
	clusters := &kubermaticv1.ExternalClusterList{}
	if err := cc.client.List(context.Background(), clusters); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list external clusters in ExternalClusterCollector: %w", err))
		return
	}

	for _, cluster := range clusters.Items {
		cc.collectCluster(ch, &cluster)
	}
}

func (cc *ExternalClusterCollector) collectCluster(ch chan<- prometheus.Metric, c *kubermaticv1.ExternalCluster) {
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

	ch <- prometheus.MustNewConstMetric(
		cc.clusterInfo,
		prometheus.GaugeValue,
		1,
		c.Name,
		c.Spec.HumanReadableName,
		string(c.Spec.CloudSpec.ProviderName),
		string(c.Status.Condition.Phase),
	)
}
