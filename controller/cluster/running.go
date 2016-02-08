package cluster

import (
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
)

func (cc *clusterController) syncRunningCluster(c *api.Cluster) (*api.Cluster, error) {
	allHealthy, health, err := cc.clusterHealth(c)
	healthChanged := false
	if err != nil {
		return nil, err
	}
	if health != nil && (c.Status.Health == nil || !reflect.DeepEqual(health, c.Status.Health)) {
		c.Status.Health = health
		c.Status.Health.LastTransitionTime = time.Now()
		healthChanged = true
	}
	if !allHealthy {
		glog.V(5).Infof("Cluster %q not healthy: %+v", c.Metadata.Name, c.Status.Health)
		if healthChanged {
			return c, nil
		}
	} else {
		glog.V(6).Infof("Cluster %q is healthy", c.Metadata.Name)
	}

	return nil, nil
}
