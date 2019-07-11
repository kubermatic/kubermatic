package cluster

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/types"
)

func (r *Reconciler) clusterHealth(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.ExtendedClusterHealth, error) {
	ns := kubernetes.NamespaceName(cluster.Name)
	extendedHealth := cluster.Status.ExtendedHealth.DeepCopy()

	type depInfo struct {
		healthStatus *kubermaticv1.HealthStatus
		minReady     int32
	}

	healthMapping := map[string]*depInfo{
		resources.ApiserverDeploymentName:             {healthStatus: &extendedHealth.Apiserver, minReady: 1},
		resources.ControllerManagerDeploymentName:     {healthStatus: &extendedHealth.Controller, minReady: 1},
		resources.SchedulerDeploymentName:             {healthStatus: &extendedHealth.Scheduler, minReady: 1},
		resources.MachineControllerDeploymentName:     {healthStatus: &extendedHealth.MachineController, minReady: 1},
		resources.OpenVPNServerDeploymentName:         {healthStatus: &extendedHealth.OpenVPN, minReady: 1},
		resources.UserClusterControllerDeploymentName: {healthStatus: &extendedHealth.UserClusterControllerManager, minReady: 1},
	}

	for name := range healthMapping {
		key := types.NamespacedName{Namespace: ns, Name: name}
		status, err := resources.HealthyDeployment(ctx, r, key, healthMapping[name].minReady)
		if err != nil {
			return nil, fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthStatus = status
	}

	var err error
	key := types.NamespacedName{Namespace: ns, Name: resources.EtcdStatefulSetName}

	etcdHealthStatus, err := resources.HealthyStatefulSet(ctx, r, key, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd health: %v", err)
	}
	extendedHealth.Etcd = etcdHealthStatus

	return extendedHealth, nil
}

func (r *Reconciler) syncHealth(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	extendedHealth, err := r.clusterHealth(ctx, cluster)
	if err != nil {
		return err
	}
	if cluster.Status.ExtendedHealth != *extendedHealth {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.ExtendedHealth = *extendedHealth
		})
	}

	return err
}
