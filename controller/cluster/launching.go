package cluster

import (
	"reflect"
	"time"

	"k8s.io/kubernetes/pkg/apis/extensions"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider/kubernetes"
)

func (cc *clusterController) clusterHealth(c *api.Cluster) (bool, *api.ClusterHealth, error) {
	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)
	deps, err := cc.depStore.ByIndex("namespace", ns)
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

	for _, obj := range deps {
		dep := obj.(*extensions.Deployment)
		role := dep.Spec.Selector.MatchLabels["role"]
		depHealth, err := cc.healthyDep(dep)
		if err != nil {
			return false, nil, err
		}
		allHealthy = allHealthy && depHealth
		if !depHealth {
			glog.V(6).Infof("Cluster %q dep %q is not healthy", c.Metadata.Name, dep.Name)
		}
		if m, found := healthMapping[role]; found {
			*m = depHealth
		}
	}

	return allHealthy, &health, nil
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
