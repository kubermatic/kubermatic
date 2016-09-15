package cluster

import (
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider/kubernetes"
	kapi "k8s.io/kubernetes/pkg/api"
)

func (cc *clusterController) clusterHealth(c *api.Cluster) (bool, *api.ClusterHealth, error) {
	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)
	rcs, err := cc.rcStore.ByIndex("namespace", ns)
	if err != nil {
		return false, nil, err
	}

	health := api.ClusterHealth{
		ClusterHealthStatus: api.ClusterHealthStatus{
			Etcd: []bool{false},
		},
	}

	healthMapping := map[string]*bool{
		"etcd": &health.Etcd[0],
		// "etcd-public" TODO(sttts): add etcd-public?
		"apiserver":          &health.Apiserver,
		"controller-manager": &health.Controller,
		"scheduler":          &health.Scheduler,
	}

	allHealthy := true

	for _, obj := range rcs {
		rc := obj.(*kapi.ReplicationController)
		role := rc.Spec.Selector["role"]
		rcHealth, err := cc.healthyRC(rc)
		if err != nil {
			return false, nil, err
		}
		allHealthy = allHealthy && rcHealth
		if !rcHealth {
			glog.V(6).Infof("Cluster %q rc %q is not healthy", c.Metadata.Name, rc.Name)
		}
		if m, found := healthMapping[role]; found {
			*m = rcHealth
		}
	}

	return allHealthy, &health, nil
}

func (cc *clusterController) syncLaunchingCluster(c *api.Cluster) (*api.Cluster, error) {
	changedC, err := cc.checkTimeout(c)
	if err != nil {
		return nil, err
	}

	// check that all replication controllers are healthy
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
