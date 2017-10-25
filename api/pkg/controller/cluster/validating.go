package cluster

import (
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

	if err = prov.Validate(c.Spec.Cloud); err != nil {
		return fmt.Errorf("cloud provider data could not be validated successfully: %v", err)
	}

	return nil
}

func (cc *controller) validatingCheckDatacenter(c *kubermaticv1.Cluster) error {
	if _, ok := cc.dcs[c.Spec.Cloud.DatacenterName]; !ok {
		return fmt.Errorf("could not find datacenter %s", c.Spec.Cloud.DatacenterName)
	}

	return nil
}
