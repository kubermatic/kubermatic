package cluster

import (
	"fmt"
	"time"

	"github.com/go-test/deep"
	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
)

func (cc *clusterController) syncRunningCluster(c *api.Cluster) (*api.Cluster, error) {
	allHealthy, health, err := cc.clusterHealth(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster heath: %v", err)
	}

	diff := deep.Equal(health.ClusterHealthStatus, c.Status.Health.ClusterHealthStatus)
	if health != nil && (c.Status.Health == nil || diff != nil) {
		glog.V(6).Infof("Updating health of cluster %q from %+v to %+v", c.Metadata.Name, c.Status.Health, health)
		c.Status.Health = health
		c.Status.Health.LastTransitionTime = time.Now()
		return c, nil
	}

	if !allHealthy {
		glog.V(5).Infof("Cluster %q not healthy: %+v", c.Metadata.Name, c.Status.Health)
	} else {
		glog.V(6).Infof("Cluster %q is healthy", c.Metadata.Name)

		if c.Spec.MasterVersion != "" {
			updateVersion, err := version.BestAutomaticUpdate(c.Spec.MasterVersion, cc.updates)
			if err != nil {
				return nil, err
			}

			if updateVersion != nil {
				// start automatic update
				c.Spec.MasterVersion = updateVersion.To
				c.Status.Phase = api.UpdatingMasterClusterStatusPhase
				c.Status.MasterUpdatePhase = api.StartMasterUpdatePhase
				c.Status.LastTransitionTime = time.Now()

			}
			return c, nil
		}
	}

	return nil, nil
}
