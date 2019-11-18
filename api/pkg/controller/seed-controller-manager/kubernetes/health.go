package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1/helper"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
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
		*healthMapping[name].healthStatus = getStatus(status, cluster)
	}

	var err error
	key := types.NamespacedName{Namespace: ns, Name: resources.EtcdStatefulSetName}

	etcdHealthStatus, err := resources.HealthyStatefulSet(ctx, r, key, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd health: %v", err)
	}
	extendedHealth.Etcd = getStatus(etcdHealthStatus, cluster)

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

	if err != nil {
		return err
	}

	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionClusterInitialized, corev1.ConditionTrue) && isClusterInitialized(cluster) {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			helper.SetClusterCondition(
				c,
				kubermaticv1.ClusterConditionClusterInitialized,
				corev1.ConditionTrue,
				"",
				"Cluster has been initialized successfully",
			)
		})
	}

	return err
}

// We assume that te cluster is still provisioning if it was not initialized fully at least once.
func getStatus(status kubermaticv1.HealthStatus, cluster *kubermaticv1.Cluster) kubermaticv1.HealthStatus {
	if status == kubermaticv1.HealthStatusDown && !isClusterInitialized(cluster) {
		return kubermaticv1.HealthStatusProvisioning
	}

	return status
}

func isClusterInitialized(cluster *kubermaticv1.Cluster) bool {
	isInitialized := cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionClusterInitialized, corev1.ConditionTrue)
	// If was set to true at least once just return true
	if isInitialized {
		return true
	}

	_, success := helper.ClusterReconciliationSuccessful(cluster)
	return success && cluster.Status.ExtendedHealth.AllHealthy()
}
