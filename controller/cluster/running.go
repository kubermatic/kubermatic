package cluster

import (
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
)

func (cc *clusterController) syncRunningCluster(c *api.Cluster) (*api.Cluster, error) {
	allHealthy, health, err := cc.clusterHealth(c)
	if err != nil {
		return nil, err
	}

	if health != nil && (c.Status.Health == nil ||
		!reflect.DeepEqual(health.ClusterHealthStatus, c.Status.Health.ClusterHealthStatus)) {
		glog.V(6).Infof("Updating health of cluster %q from %+v to %+v", c.Metadata.Name, c.Status.Health, health)
		c.Status.Health = health
		c.Status.Health.LastTransitionTime = time.Now()
		return c, nil
	}
	if !allHealthy {
		glog.V(5).Infof("Cluster %q not healthy: %+v", c.Metadata.Name, c.Status.Health)
	} else {
		glog.V(6).Infof("Cluster %q is healthy", c.Metadata.Name)
	}

	return nil, nil
}
