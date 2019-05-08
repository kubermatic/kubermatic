package packet

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

const (
	defaultBillingCycle = "hourly"
)

type packet struct {
	dcs map[string]provider.DatacenterMeta
}

// NewCloudProvider creates a new packet provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &packet{
		dcs: dcs,
	}
}

// DefaultCloudSpec adds defaults to the CloudSpec.
func (p *packet) DefaultCloudSpec(spec kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec validates the given CloudSpec.
func (p *packet) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	if spec.Packet.APIKey == "" {
		return fmt.Errorf("apiKey cannot be empty")
	}
	if spec.Packet.ProjectID == "" {
		return fmt.Errorf("projectID cannot be empty")
	}
	return nil
}

// InitializeCloudProvider initializes a cluster, in particular
// updates BillingCycle to the defaultBillingCycle, if it is not set.
func (p *packet) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error
	dc, ok := p.dcs[cluster.Spec.Cloud.DatacenterName]
	if !ok {
		return nil, fmt.Errorf("could not find datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	if dc.Spec.Packet == nil {
		return nil, fmt.Errorf("datacenter %q is not a valid Packet datacenter", cluster.Spec.Cloud.DatacenterName)
	}

	if cluster.Spec.Cloud.Packet.BillingCycle == "" {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Packet.BillingCycle = defaultBillingCycle
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// CleanUpCloudProvider
func (p *packet) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}
