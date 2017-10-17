package cluster

import (
	"fmt"

	"github.com/go-test/deep"
	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cc *controller) syncRunningCluster(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	allHealthy, health, err := cc.clusterHealth(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster heath: %v", err)
	}

	diff := deep.Equal(health.ClusterHealthStatus, c.Status.Health.ClusterHealthStatus)
	if health != nil && (c.Status.Health == nil || diff != nil) {
		glog.V(6).Infof("Updating health of cluster %q from %+v to %+v", c.Name, c.Status.Health, health)
		c.Status.Health = health
		c.Status.Health.LastTransitionTime = metav1.Now()
		return c, nil
	}

	if !allHealthy {
		glog.V(5).Infof("Cluster %q not healthy: %+v", c.Name, c.Status.Health)
	} else {
		glog.V(6).Infof("Cluster %q is healthy", c.Name)

		if c.Spec.MasterVersion != "" {
			updateVersion, err := version.BestAutomaticUpdate(c.Spec.MasterVersion, cc.updates)
			if err != nil {
				return nil, err
			}

			if updateVersion != nil {
				// start automatic update
				c.Spec.MasterVersion = updateVersion.To
				c.Status.Phase = kubermaticv1.UpdatingMasterClusterStatusPhase
				c.Status.MasterUpdatePhase = kubermaticv1.StartMasterUpdatePhase
				c.Status.LastTransitionTime = metav1.Now()

			}
			return c, nil
		}
	}

	return nil, nil
}
