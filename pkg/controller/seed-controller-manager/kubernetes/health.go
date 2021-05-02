/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubernetes

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
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
		*healthMapping[name].healthStatus = kubermaticv1helper.GetHealthStatus(status, cluster, r.versions)
	}

	var err error
	key := types.NamespacedName{Namespace: ns, Name: resources.EtcdStatefulSetName}

	etcdHealthStatus, err := resources.HealthyStatefulSet(ctx, r, key, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd health: %v", err)
	}
	extendedHealth.Etcd = kubermaticv1helper.GetHealthStatus(etcdHealthStatus, cluster, r.versions)

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
	// set ClusterConditionEtcdClusterInitialized, this should be done only once
	// when etcd becomes healthy for the first time.
	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEtcdClusterInitialized, corev1.ConditionTrue) && extendedHealth.Etcd == kubermaticv1.HealthStatusUp {
		if err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			kubermaticv1helper.SetClusterCondition(
				c,
				r.versions,
				kubermaticv1.ClusterConditionEtcdClusterInitialized,
				corev1.ConditionTrue,
				"",
				"Etcd Cluster has been initialized successfully",
			)
		}); err != nil {
			return fmt.Errorf("failed to sec cluster %s condition: %v", kubermaticv1.ClusterConditionEtcdClusterInitialized, err)
		}
	}

	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionClusterInitialized, corev1.ConditionTrue) && kubermaticv1helper.IsClusterInitialized(cluster, r.versions) {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			kubermaticv1helper.SetClusterCondition(
				c,
				r.versions,
				kubermaticv1.ClusterConditionClusterInitialized,
				corev1.ConditionTrue,
				"",
				"Cluster has been initialized successfully",
			)
		})
	}

	return err
}
