package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

func (cc *controller) validateCluster(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if err := cc.validateDatacenter(c); err != nil {
		return nil, fmt.Errorf("failed to validate datacenter: %v", err)
	}

	if err := cc.validateCloudSpec(c); err != nil {
		return nil, fmt.Errorf("failed to validate cloud spec: %v", err)
	}

	return c, nil
}

func (cc *controller) validateCloudSpec(c *kubermaticv1.Cluster) error {
	_, prov, err := provider.ClusterCloudProvider(cc.cps, c)
	if err != nil {
		return err
	}

	if err = prov.ValidateCloudSpec(c.Spec.Cloud); err != nil {
		return fmt.Errorf("cloud provider data could not be validated successfully: %v", err)
	}

	return nil
}

func (cc *controller) validateDatacenter(c *kubermaticv1.Cluster) error {
	//Validate seed datacenter
	seedDc, found := cc.dcs[c.Spec.SeedDatacenterName]
	if !found {
		return fmt.Errorf("could not find given seed datacenter %q", c.Spec.SeedDatacenterName)
	}
	if !seedDc.IsSeed {
		return fmt.Errorf("given seed datacenter %q is not configured as a seed datacenter", c.Spec.SeedDatacenterName)
	}

	//Validate node datacenter
	dc, found := cc.dcs[c.Spec.Cloud.DatacenterName]
	if !found {
		return fmt.Errorf("could not find node datacenter %q", c.Spec.Cloud.DatacenterName)
	}
	if dc.IsSeed {
		return fmt.Errorf("given datacenter %q is not configured as a node datacenter", c.Spec.SeedDatacenterName)
	}

	return nil
}
