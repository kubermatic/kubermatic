package cluster

import (
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/pkg/extensions/etcd"
	"github.com/kubermatic/api/pkg/provider/kubernetes"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (cc *clusterController) clusterHealth(c *api.Cluster) (bool, *api.ClusterHealth, error) {
	ns := kubernetes.NamespaceName(c.Metadata.Name)
	deps, err := cc.depStore.ByIndex("namespace", ns)
	if err != nil {
		return false, nil, err
	}

	etcds, err := cc.etcdClusterStore.ByIndex("namespace", ns)
	if err != nil {
		return false, nil, err
	}

	health := api.ClusterHealth{
		ClusterHealthStatus: api.ClusterHealthStatus{
			Etcd: []bool{false},
		},
	}

	healthMapping := map[string]*bool{
		"etcd-cluster":       &health.Etcd[0],
		"apiserver":          &health.Apiserver,
		"controller-manager": &health.Controller,
		"scheduler":          &health.Scheduler,
	}

	allHealthy := true

	for _, obj := range deps {
		dep := obj.(*v1beta1.Deployment)
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

	for _, obj := range etcds {
		etcd := obj.(*etcd.Cluster)

		etcdHealth, err := cc.healthyEtcd(etcd)
		if err != nil {
			return false, nil, err
		}
		allHealthy = allHealthy && etcdHealth
		if !etcdHealth {
			glog.V(6).Infof("Cluster %q etcd %q is not healthy", c.Metadata.Name, etcd.Metadata.Name)
		}
		if m, found := healthMapping[etcd.Metadata.Name]; found {
			*m = etcdHealth
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
