package cluster

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/types"
)

func (r *Reconciler) clusterHealth(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.ClusterHealth, error) {
	ns := kubernetes.NamespaceName(cluster.Name)
	health := cluster.Status.Health.DeepCopy()

	type depInfo struct {
		healthy  *bool
		minReady int32
	}

	healthMapping := map[string]*depInfo{
		resources.ApiserverDeploymentName:             {healthy: &health.Apiserver, minReady: 1},
		resources.ControllerManagerDeploymentName:     {healthy: &health.Controller, minReady: 1},
		resources.SchedulerDeploymentName:             {healthy: &health.Scheduler, minReady: 1},
		resources.MachineControllerDeploymentName:     {healthy: &health.MachineController, minReady: 1},
		resources.OpenVPNServerDeploymentName:         {healthy: &health.OpenVPN, minReady: 1},
		resources.UserClusterControllerDeploymentName: {healthy: &health.UserClusterControllerManager, minReady: 1},
	}

	for name := range healthMapping {
		key := types.NamespacedName{Namespace: ns, Name: name}
		healthy, err := resources.HealthyDeployment(ctx, r, key, healthMapping[name].minReady)
		if err != nil {
			return nil, fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthy = healthy
	}

	var err error
	key := types.NamespacedName{Namespace: ns, Name: resources.EtcdStatefulSetName}
	health.Etcd, err = resources.HealthyStatefulSet(ctx, r, key, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd health: %v", err)
	}

	return health, nil
}

func (r *Reconciler) syncHealth(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	health, err := r.clusterHealth(ctx, cluster)
	if err != nil {
		return err
	}
	if cluster.Status.Health != *health {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.Health = *health
		})
	}

	return err
}
