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

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/prometheus/client_golang/prometheus"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	addonPrefix = "kubermatic_addon_"
)

// AddonCollector exports metrics for addon resources
type AddonCollector struct {
	client ctrlruntimeclient.Reader

	addonCreated *prometheus.Desc
	addonDeleted *prometheus.Desc
}

// MustRegisterAddonCollector registers the addon collector at the given prometheus registry
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
	}

	registry.MustRegister(cc)
}

// Describe returns the metrics descriptors
func (cc AddonCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cc.addonCreated
	ch <- cc.addonDeleted
}

// Collect gets called by prometheus to collect the metrics
func (cc AddonCollector) Collect(ch chan<- prometheus.Metric) {
	addons := &kubermaticv1.AddonList{}
	if err := cc.client.List(
		context.Background(),
		addons,
		&ctrlruntimeclient.ListOptions{}); err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to list addons in AddonCollector: %v", err))
		return
	}
	for _, addon := range addons.Items {
		cc.collectAddon(ch, &addon)
	}
}

func (cc *AddonCollector) collectAddon(ch chan<- prometheus.Metric, addon *kubermaticv1.Addon) {
	if len(addon.OwnerReferences) < 1 || addon.OwnerReferences[0].Kind != kubermaticv1.ClusterKindName {
		utilruntime.HandleError(fmt.Errorf("No owning cluster for addon %v/%v", addon.Namespace, addon.Name))
	}

	clusterName := addon.OwnerReferences[0].Name

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
