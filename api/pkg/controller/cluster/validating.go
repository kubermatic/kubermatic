package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cc *controller) syncValidatingCluster(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	//Should be called last due to increased number of external api usage on some cloud providers
	err := cc.validatingCheckCloudSpec(c)
	if err != nil {
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

	err = prov.Validate(c.Spec.Cloud)
	if err != nil {
		return fmt.Errorf("cloud provider data could not be validated successfully: %v", err)
	}

	return nil
}
