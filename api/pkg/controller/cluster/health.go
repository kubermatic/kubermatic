package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"k8s.io/apimachinery/pkg/api/errors"
)

func (cc *Controller) clusterHealth(c *kubermaticv1.Cluster) (*kubermaticv1.ClusterHealth, error) {
	ns := kubernetes.NamespaceName(c.Name)
	health := kubermaticv1.ClusterHealth{
		ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{},
	}

	type depInfo struct {
		healthy  *bool
		minReady int32
	}

	healthMapping := map[string]*depInfo{
		resources.ApiserverDeploymentName:         {healthy: &health.Apiserver, minReady: 1},
		resources.ControllerManagerDeploymentName: {healthy: &health.Controller, minReady: 1},
		resources.SchedulerDeploymentName:         {healthy: &health.Scheduler, minReady: 1},
		resources.MachineControllerDeploymentName: {healthy: &health.MachineController, minReady: 1},
		resources.OpenVPNServerDeploymentName:     {healthy: &health.OpenVPN, minReady: 1},
	}

	for name := range healthMapping {
		healthy, err := cc.healthyDeployment(ns, name, healthMapping[name].minReady)
		if err != nil {
			return nil, fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthy = healthy
	}

	var err error
	health.Etcd, err = cc.healthyStatefulSet(ns, resources.EtcdStatefulSetName, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd health: %v", err)
	}

	return &health, nil
}

func (cc *Controller) healthyDeployment(ns, name string, minReady int32) (bool, error) {
	dep, err := cc.deploymentLister.Deployments(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return dep.Status.ReadyReplicas >= minReady, nil
}

func (cc *Controller) healthyStatefulSet(ns, name string, minReady int32) (bool, error) {
	set, err := cc.statefulSetLister.StatefulSets(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return set.Status.ReadyReplicas >= minReady, nil
}

func (cc *Controller) syncHealth(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	health, err := cc.clusterHealth(c)
	if err != nil {
		return nil, err
	}
	if c.Status.Health != *health {
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Status.Health = *health
		})
	}

	return c, err
}
