package cluster

import (
	"time"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

// UpdateTimeout represent the duration to wait before considering an update failed
const UpdateTimeout = time.Minute * 30

func (cc *controller) syncUpdatingClusterMaster(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if c.Status.MasterUpdatePhase == kubermaticv1.FinishMasterUpdatePhase {
		c.Status.MasterUpdatePhase = ""
		c.Status.Phase = kubermaticv1.RunningClusterStatusPhase
		return c, nil
	}

	if time.Now().After(c.Status.LastTransitionTime.Add(UpdateTimeout)) {
		if c.Status.LastDeployedMasterVersion == c.Spec.MasterVersion {
			// Rollback failed, failed cluster
			c.Status.Phase = kubermaticv1.FailedClusterStatusPhase
		} else {
			// Initiate Rollback
			c.Spec.MasterVersion = c.Status.LastDeployedMasterVersion
			c.Status.MasterUpdatePhase = kubermaticv1.StartMasterUpdatePhase
		}
		return c, nil
	}

	return cc.updateController.Sync(c)
}
