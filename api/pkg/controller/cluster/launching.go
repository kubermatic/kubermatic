package cluster

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
)

func (cc *clusterController) clusterHealth(c *api.Cluster) (bool, *api.ClusterHealth, error) {
	ns := kubernetes.NamespaceName(c.Metadata.Name)
	health := api.ClusterHealth{
		ClusterHealthStatus: api.ClusterHealthStatus{},
	}

	type depInfo struct {
		healthy  *bool
		minReady int32
	}

	healthMapping := map[string]*depInfo{
		"apiserver":          &depInfo{healthy: &health.Apiserver, minReady: 1},
		"controller-manager": &depInfo{healthy: &health.Controller, minReady: 1},
		"scheduler":          &depInfo{healthy: &health.Scheduler, minReady: 1},
		"node-controller":    &depInfo{healthy: &health.NodeController, minReady: 1},
	}

	for name := range healthMapping {
		healthy, err := cc.healthyDep(ns, name, healthMapping[name].minReady)
		if err != nil {
			return false, nil, fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthy = healthy
	}

	var err error
	health.Etcd, err = cc.healthyEtcd(ns, resources.EtcdClusterName)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get etcd health: %v", err)
	}

	return health.AllHealthy(), &health, nil
}

func (cc *clusterController) syncLaunchingCluster(c *api.Cluster) (*api.Cluster, error) {
	changedC, err := cc.checkTimeout(c)
	if err != nil {
		return nil, err
	}

	// check that all deployments are healthy
	allHealthy, health, err := cc.clusterHealth(c)
	if err != nil {
		return nil, err
	}

	if health != nil && (c.Status.Health == nil ||
		!reflect.DeepEqual(health.ClusterHealthStatus, c.Status.Health.ClusterHealthStatus)) {
		glog.V(6).Infof("Updating health of cluster %q from %+v to %+v", c.Metadata.Name, c.Status.Health, health)
		c.Status.Health = health
		c.Status.Health.LastTransitionTime = time.Now()
		changedC = c
	}

	if !allHealthy {
		glog.V(5).Infof("Cluster %q not yet healthy: %+v", c.Metadata.Name, c.Status.Health)
		return changedC, nil
	}

	// no error until now? We are running.
	c.Status.Phase = api.RunningClusterStatusPhase
	c.Status.LastTransitionTime = time.Now()

	return c, nil
}
