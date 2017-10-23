package cluster

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cc *controller) syncValidatingCluster(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	_, prov, err := provider.ClusterCloudProvider(cc.cps, c)
	if err != nil {
		return nil, err
	}

	err = prov.Validate(c.Spec.Cloud)
	if err != nil {
		return nil, err
	}

	c.Status.Phase = kubermaticv1.PendingClusterStatusPhase
	c.Status.LastTransitionTime = metav1.Now()

	return c, nil
}
