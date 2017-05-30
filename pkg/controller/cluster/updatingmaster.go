package cluster

import (
	"time"

	api "github.com/kubermatic/api/pkg/types"
)

// UpdateTimeout represent the duration to wait before considering an update failed
const UpdateTimeout = time.Minute * 30

func (cc *clusterController) syncUpdatingClusterMaster(c *api.Cluster) (*api.Cluster, error) {
	if c.Status.MasterUpdatePhase == api.FinishMasterUpdatePhase {
		c.Status.MasterUpdatePhase = ""
		c.Status.Phase = api.RunningClusterStatusPhase
		return c, nil
	}

	if time.Now().After(c.Status.LastTransitionTime.Add(UpdateTimeout)) {
		if c.Status.LastDeployedMasterVersion == c.Spec.MasterVersion {
			// Rollback failed, fail cluster
			c.Status.Phase = api.FailedClusterStatusPhase
		} else {
			// Initiate Rollback
			c.Spec.MasterVersion = c.Status.LastDeployedMasterVersion
			c.Status.MasterUpdatePhase = api.StartMasterUpdatePhase
		}
		return c, nil
	}

	return cc.updateController.Sync(c)
}
