package cluster

import (
	"time"

	"github.com/kubermatic/api"
)

const clusterUpdateTimeout = time.Minute * 30

func (cc *clusterController) syncUpdatingClusterMaster(c *api.Cluster) (*api.Cluster, error) {

	if c.Status.MasterUpdatePhase == api.FinishMasterUpdatePhase {
		c.Status.MasterUpdatePhase = ""
		c.Status.Phase = api.RunningClusterStatusPhase
		return c, nil
	}

	if time.Now().After(c.Status.LastTransitionTime.Add(clusterUpdateTimeout)) {
		if c.Status.LastDeployedMasterVersion == c.Spec.TargetMasterVersion {
			// Rollback failed, fail cluster
			c.Status.Phase = api.FailedClusterStatusPhase
		} else {
			// Initiate Rollback
			c.Spec.TargetMasterVersion = c.Status.LastDeployedMasterVersion
			c.Status.MasterUpdatePhase = api.StartMasterUpdatePhase
		}
		return c, nil
	}

	return c, nil
}
