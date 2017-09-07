package cluster

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/controller/resources"
	"github.com/kubermatic/kubermatic/api/provider/kubernetes"
)

func (cc *clusterController) clusterHealth(c *api.Cluster) (bool, *api.ClusterHealth, error) {
	ns := kubernetes.NamespaceName(c.Metadata.Name)
	health := api.ClusterHealth{
		ClusterHealthStatus: api.ClusterHealthStatus{},
	}

	healthMapping := map[string]*bool{
		"apiserver":          &health.Apiserver,
		"controller-manager": &health.Controller,
		"scheduler":          &health.Scheduler,
		"node-controller":    &health.NodeController,
	}

	for name := range healthMapping {
		healthy, err := cc.healthyDep(ns, name)
		if err != nil {
			return false, nil, fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name] = healthy
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
