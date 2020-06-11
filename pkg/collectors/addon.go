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
