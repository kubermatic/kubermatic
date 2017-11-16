package cluster

import (
	"errors"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cc *controller) syncValidatingCluster(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if err := cc.validatingCheckDatacenter(c); err != nil {
		return nil, err
	}

	//Should be called last due to increased number of external api usage on some cloud providers
	if err := cc.validatingCheckCloudSpec(c); err != nil {
		return nil, err
	}

	c.Status.Phase = kubermaticv1.PendingClusterStatusPhase
	c.Status.LastTransitionTime = metav1.Now()

	return c, nil
}

func (cc *controller) validatingCheckCloudSpec(c *kubermaticv1.Cluster) error {
	_, prov, err := provider.ClusterCloudProvider(cc.cps, c)
	if err != nil {
		return err
	}

	if err = prov.ValidateCloudSpec(c.Spec.Cloud); err != nil {
		return fmt.Errorf("cloud provider data could not be validated successfully: %v", err)
	}

	return nil
}

func (cc *controller) validatingCheckDatacenter(c *kubermaticv1.Cluster) error {
	//Validate seed datacenter
	seedDc, found := cc.dcs[c.Spec.SeedDatacenterName]
	if !found {
		return fmt.Errorf("could not find given seed datacenter %q", c.Spec.SeedDatacenterName)
	}
	if !seedDc.IsSeed {
		return fmt.Errorf("given seed datacenter %q is not configured as a seed datacenter", c.Spec.SeedDatacenterName)
	}

	// If we have BringYourOwn, than no node provider must be set
	if c.Spec.Cloud.BringYourOwn != nil {
		if c.Spec.Cloud.DatacenterName != "" {
			return errors.New("node dc is not allowed when using bringyourown")
		}
	} else {
		//Validate node datacenter
		dc, found := cc.dcs[c.Spec.Cloud.DatacenterName]
		if !found {
			return fmt.Errorf("could not find node datacenter %q", c.Spec.Cloud.DatacenterName)
		}
		if dc.IsSeed {
			return fmt.Errorf("given datacenter %q is not configured as a node datacenter", c.Spec.SeedDatacenterName)
		}
	}

	return nil
}
