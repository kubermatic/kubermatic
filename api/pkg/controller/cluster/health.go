package cluster

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/types"
)

func (r *Reconciler) clusterHealth(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.ClusterHealth, *kubermaticv1.ExtendedClusterHealth, error) {
	ns := kubernetes.NamespaceName(cluster.Name)
	health := cluster.Status.Health.DeepCopy()
	extendedHealth := cluster.Status.ExtendedHealth.DeepCopy()

	type depInfo struct {
		healthy      *bool
		healthStatus *kubermaticv1.HealthStatus
		minReady     int32
	}

	healthMapping := map[string]*depInfo{
		resources.ApiserverDeploymentName:             {healthy: &health.Apiserver, healthStatus: &extendedHealth.Apiserver, minReady: 1},
		resources.ControllerManagerDeploymentName:     {healthy: &health.Controller, healthStatus: &extendedHealth.Controller, minReady: 1},
		resources.SchedulerDeploymentName:             {healthy: &health.Scheduler, healthStatus: &extendedHealth.Scheduler, minReady: 1},
		resources.MachineControllerDeploymentName:     {healthy: &health.MachineController, healthStatus: &extendedHealth.MachineController, minReady: 1},
		resources.OpenVPNServerDeploymentName:         {healthy: &health.OpenVPN, healthStatus: &extendedHealth.OpenVPN, minReady: 1},
		resources.UserClusterControllerDeploymentName: {healthy: &health.UserClusterControllerManager, healthStatus: &extendedHealth.UserClusterControllerManager, minReady: 1},
	}

	for name := range healthMapping {
		key := types.NamespacedName{Namespace: ns, Name: name}
		status, err := resources.HealthyDeployment(ctx, r, key, healthMapping[name].minReady)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthy = isRunning(status)
		*healthMapping[name].healthStatus = status
	}

	var err error
	key := types.NamespacedName{Namespace: ns, Name: resources.EtcdStatefulSetName}

	etcdHealthStatus, err := resources.HealthyStatefulSet(ctx, r, key, 2)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get etcd health: %v", err)
	}
	health.Etcd = isRunning(etcdHealthStatus)
	extendedHealth.Etcd = etcdHealthStatus

	return health, extendedHealth, nil
}

func isRunning(status kubermaticv1.HealthStatus) bool {
	return status == kubermaticv1.UP
}

func (r *Reconciler) syncHealth(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	health, extendedHealth, err := r.clusterHealth(ctx, cluster)
	if err != nil {
		return err
	}
	if cluster.Status.Health != *health {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.Health = *health
			c.Status.ExtendedHealth = *extendedHealth
		})
	}

	return err
}
