/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	seedPrefix = "kubermatic_seed_"
)

// SeedCollector exports metrics for seed resources.
type SeedCollector struct {
	client ctrlruntimeclient.Reader

	seedInfo      *prometheus.Desc
	seedClusters  *prometheus.Desc
	seedCondition *prometheus.Desc
}

// MustRegisterSeedCollector registers the seed collector at the given prometheus registry.
func MustRegisterSeedCollector(registry prometheus.Registerer, client ctrlruntimeclient.Reader) {
	cc := &SeedCollector{
		client: client,
		seedInfo: prometheus.NewDesc(
			seedPrefix+"info",
			"Additional seed information",
			[]string{
				"seed_name",
				"country",
				"location",
				"phase",
				"kubermatic_version",
				"kubernetes_version",
			},
			nil,
		),
		seedClusters: prometheus.NewDesc(
			seedPrefix+"clusters",
			"Number of user clusters per seed cluster",
			[]string{"seed_name"},
			nil,
		),
		seedCondition: prometheus.NewDesc(
			seedPrefix+"condition",
			"Binary metric that describes one of the Seed conditions",
			[]string{
				"seed_name",
				"condition",
				"reason",
			},
			nil,
		),
	}

	registry.MustRegister(cc)
}

// Describe returns the metrics descriptors.
func (cc SeedCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(cc, ch)
}

// Collect gets called by prometheus to collect the metrics.
func (cc SeedCollector) Collect(ch chan<- prometheus.Metric) {
	seeds := &kubermaticv1.SeedList{}
	if err := cc.client.List(context.Background(), seeds); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list seeds in SeedCollector: %w", err))
		return
	}

	kubernetesLabelSet := sets.New[string]()
	for _, seed := range seeds.Items {
		kubernetesLabelSet = kubernetesLabelSet.Union(sets.KeySet(seed.Labels))
	}

	kubernetesLabels := caseInsensitiveSort(sets.List(kubernetesLabelSet))

	prometheusLabels := convertToPrometheusLabels(kubernetesLabels)
	labelsGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: seedPrefix + "labels",
		Help: "Kubernetes labels on Seed resources",
	}, append([]string{"seed_name"}, prometheusLabels...))

	for _, seed := range seeds.Items {
		cc.collectSeed(ch, &seed, kubernetesLabels, labelsGauge)
	}
}

func (cc *SeedCollector) collectSeed(ch chan<- prometheus.Metric, seed *kubermaticv1.Seed, kubernetesLabels []string, labelsGaugeVec *prometheus.GaugeVec) {
	ch <- prometheus.MustNewConstMetric(
		cc.seedInfo,
		prometheus.GaugeValue,
		1,
		seed.Name,
		seed.Spec.Country,
		seed.Spec.Location,
		string(seed.Status.Phase),
		seed.Status.Versions.Kubermatic,
		seed.Status.Versions.Cluster,
	)

	ch <- prometheus.MustNewConstMetric(
		cc.seedClusters,
		prometheus.GaugeValue,
		float64(seed.Status.Clusters),
		seed.Name,
	)

	for condName, cond := range seed.Status.Conditions {
		value := 0
		if cond.Status == corev1.ConditionTrue {
			value = 1
		}

		ch <- prometheus.MustNewConstMetric(
			cc.seedCondition,
			prometheus.GaugeValue,
			float64(value),
			seed.Name,
			string(condName),
			cond.Reason,
		)
	}

	// assemble the labels for this seed, in the order given by kubernetesLabels, but
	// taking special care of label key conflicts
	seedLabels := []string{seed.Name}
	usedLabels := sets.New[string]()
	for _, key := range kubernetesLabels {
		prometheusLabel := convertToPrometheusLabel(key)
		if !usedLabels.Has(prometheusLabel) {
			seedLabels = append(seedLabels, seed.Labels[key])
			usedLabels.Insert(prometheusLabel)
		}
	}

	labelsGaugeVec.WithLabelValues(seedLabels...).Collect(ch)
}
