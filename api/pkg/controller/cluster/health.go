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
		healthy *bool
	}

	healthMapping := map[string]*depInfo{
		resources.ApiserverDeploymentName:         {healthy: &health.Apiserver},
		resources.ControllerManagerDeploymentName: {healthy: &health.Controller},
		resources.SchedulerDeploymentName:         {healthy: &health.Scheduler},
		resources.MachineControllerDeploymentName: {healthy: &health.MachineController},
		resources.OpenVPNServerDeploymentName:     {healthy: &health.OpenVPN},
	}

	for name := range healthMapping {
		healthy, err := cc.healthyDeployment(ns, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthy = healthy
	}

	var err error
	health.Etcd, err = cc.healthyStatefulSet(ns, resources.EtcdStatefulSetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd health: %v", err)
	}

	return &health, nil
}

func (cc *Controller) healthyDeployment(ns, name string) (bool, error) {
	dep, err := cc.deploymentLister.Deployments(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return dep.Status.ReadyReplicas == *dep.Spec.Replicas && dep.Status.UpdatedReplicas == *dep.Spec.Replicas, nil
}

func (cc *Controller) healthyStatefulSet(ns, name string) (bool, error) {
	set, err := cc.statefulSetLister.StatefulSets(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return set.Status.ReadyReplicas == *set.Spec.Replicas && set.Status.CurrentReplicas == *set.Spec.Replicas, nil
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
