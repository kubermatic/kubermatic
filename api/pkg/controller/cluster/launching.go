package cluster

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cc *controller) clusterHealth(c *kubermaticv1.Cluster) (bool, *kubermaticv1.ClusterHealth, error) {
	ns := kubernetes.NamespaceName(c.Name)
	health := kubermaticv1.ClusterHealth{
		ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{},
	}

	type depInfo struct {
		healthy  *bool
		minReady int32
	}

	healthMapping := map[string]*depInfo{
		"apiserver":          {healthy: &health.Apiserver, minReady: 1},
		"controller-manager": {healthy: &health.Controller, minReady: 1},
		"scheduler":          {healthy: &health.Scheduler, minReady: 1},
		"node-controller":    {healthy: &health.NodeController, minReady: 1},
	}

	for name := range healthMapping {
		healthy, err := cc.healthyDep(c.Spec.SeedDatacenterName, ns, name, healthMapping[name].minReady)
		if err != nil {
			return false, nil, fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthy = healthy
	}

	var err error
	health.Etcd, err = cc.healthyEtcd(c.Spec.SeedDatacenterName, ns, resources.EtcdClusterName)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get etcd health: %v", err)
	}

	return health.AllHealthy(), &health, nil
}

func (cc *controller) launchingClusterHealth(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	// check that all deployments are healthy
	allHealthy, health, err := cc.clusterHealth(c)
	if err != nil {
		return nil, err
	}

	if health != nil && (c.Status.Health == nil ||
		!reflect.DeepEqual(health.ClusterHealthStatus, c.Status.Health.ClusterHealthStatus)) {
		glog.V(6).Infof("Updating health of cluster %q from %+v to %+v", c.Name, c.Status.Health, health)
		c.Status.Health = health
		c.Status.Health.LastTransitionTime = metav1.Now()
	}

	if !allHealthy {
		glog.V(5).Infof("Cluster %q not yet healthy: %+v", c.Name, c.Status.Health)
		return c, nil
	}

	return nil, nil
}

// launchingClusterReachable checks if the cluster is reachable via its external name
func (cc *controller) launchingClusterReachable(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	client, err := c.GetClient()
	if err != nil {
		return nil, err
	}
	_, err = client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		glog.V(5).Infof("Cluster %q not yet reachable: %v", c.Name, err)
		return c, nil
	}

	return nil, nil
}

func (cc *controller) syncLaunchingCluster(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {

	changedC, err := cc.launchingClusterHealth(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	changedC, err = cc.launchingClusterReachable(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// no error until now? We are running.
	c.Status.Phase = kubermaticv1.RunningClusterStatusPhase
	c.Status.LastTransitionTime = metav1.Now()

	return c, nil
}
