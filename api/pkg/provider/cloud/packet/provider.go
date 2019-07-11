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
	dcs map[string]*kubermaticv1.SeedDatacenter
}

// NewCloudProvider creates a new packet provider.
func NewCloudProvider(dcs map[string]*kubermaticv1.SeedDatacenter) provider.CloudProvider {
	return &packet{
		dcs: dcs,
	}
}

// DefaultCloudSpec adds defaults to the CloudSpec.
func (p *packet) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
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
	nodeDC, err := provider.NodeLocationFromSeedMap(p.dcs, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, err
	}

	if nodeDC.Packet == nil {
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

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (p *packet) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}
