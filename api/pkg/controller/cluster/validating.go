package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

func (cc *Controller) validateCluster(c *kubermaticv1.Cluster) error {
	if err := cc.validateDatacenter(c); err != nil {
		return fmt.Errorf("failed to validate datacenter: %v", err)
	}

	if err := cc.validateCloudSpec(c); err != nil {
		return fmt.Errorf("failed to validate cloud spec: %v", err)
	}

	return nil
}

func (cc *Controller) validateCloudSpec(c *kubermaticv1.Cluster) error {
	_, prov, err := provider.ClusterCloudProvider(cc.cps, c)
	if err != nil {
		return err
	}

	if err = prov.ValidateCloudSpec(c.Spec.Cloud); err != nil {
		return fmt.Errorf("cloud provider data could not be validated successfully: %v", err)
	}

	return nil
}

func (cc *Controller) validateDatacenter(c *kubermaticv1.Cluster) error {
	//Validate node datacenter
	dc, found := cc.dcs[c.Spec.Cloud.DatacenterName]
	if !found {
		return fmt.Errorf("could not find node datacenter %q", c.Spec.Cloud.DatacenterName)
	}
	if dc.IsSeed {
		return fmt.Errorf("specified node datacenter %q is not configured as a seed-datacenter", c.Spec.Cloud.DatacenterName)
	}

	return nil
}
