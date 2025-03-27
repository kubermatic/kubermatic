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
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	addonPrefix = "kubermatic_addon_"
)

// AddonCollector exports metrics for addon resources.
type AddonCollector struct {
	client ctrlruntimeclient.Reader

	addonCreated       *prometheus.Desc
	addonDeleted       *prometheus.Desc
	addonReconcileFail *prometheus.Desc
}

// MustRegisterAddonCollector registers the addon collector at the given prometheus registry.
func MustRegisterAddonCollector(registry prometheus.Registerer, client ctrlruntimeclient.Reader) {
	cc := &AddonCollector{
		client: client,
		addonCreated: prometheus.NewDesc(
			addonPrefix+"created",
			"Unix creation timestamp",
			[]string{"cluster", "addon"},
			nil,
		),
		addonDeleted: prometheus.NewDesc(
			addonPrefix+"deleted",
			"Unix deletion timestamp",
			[]string{"cluster", "addon"},
			nil,
		),
		addonReconcileFail: prometheus.NewDesc(
			addonPrefix+"reconcile_failed",
			"Reconcile is failing",
			[]string{"cluster", "addon"},
			nil,
		),
	}

	registry.MustRegister(cc)
}

// Describe returns the metrics descriptors.
func (cc AddonCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cc.addonCreated
	ch <- cc.addonDeleted
	ch <- cc.addonReconcileFail
}

// Collect gets called by prometheus to collect the metrics.
func (cc AddonCollector) Collect(ch chan<- prometheus.Metric) {
	addons := &kubermaticv1.AddonList{}
	if err := cc.client.List(
		context.Background(),
		addons,
		&ctrlruntimeclient.ListOptions{}); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list addons in AddonCollector: %w", err))
		return
	}
	for _, addon := range addons.Items {
		cc.collectAddon(ch, &addon)
	}
}

func (cc *AddonCollector) collectAddon(ch chan<- prometheus.Metric, addon *kubermaticv1.Addon) {
	parts := strings.Split(addon.Namespace, "-")
	clusterName := parts[1]

	notCreated := 1
	if addon.Status.Conditions[kubermaticv1.AddonResourcesCreated].Status == corev1.ConditionTrue {
		notCreated = 0
	}

	ch <- prometheus.MustNewConstMetric(
		cc.addonReconcileFail,
		prometheus.GaugeValue,
		float64(notCreated),
		clusterName,
		addon.Name,
	)

	ch <- prometheus.MustNewConstMetric(
		cc.addonCreated,
		prometheus.GaugeValue,
		float64(addon.CreationTimestamp.Unix()),
		clusterName,
		addon.Name,
	)

	if addon.DeletionTimestamp != nil {
		ch <- prometheus.MustNewConstMetric(
			cc.addonDeleted,
			prometheus.GaugeValue,
			float64(addon.DeletionTimestamp.Unix()),
			clusterName,
			addon.Name,
		)
	}
}
