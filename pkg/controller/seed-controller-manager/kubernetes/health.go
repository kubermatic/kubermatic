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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (r *Reconciler) clusterHealth(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.ExtendedClusterHealth, error) {
	ns := cluster.Status.NamespaceName
	extendedHealth := cluster.Status.ExtendedHealth.DeepCopy()

	type depInfo struct {
		healthStatus *kubermaticv1.HealthStatus
		minReady     int32
	}

	healthMapping := map[string]*depInfo{
		resources.ApiserverDeploymentName:             {healthStatus: &extendedHealth.Apiserver, minReady: 1},
		resources.ControllerManagerDeploymentName:     {healthStatus: &extendedHealth.Controller, minReady: 1},
		resources.SchedulerDeploymentName:             {healthStatus: &extendedHealth.Scheduler, minReady: 1},
		resources.OpenVPNServerDeploymentName:         {healthStatus: &extendedHealth.OpenVPN, minReady: 1},
		resources.UserClusterControllerDeploymentName: {healthStatus: &extendedHealth.UserClusterControllerManager, minReady: 1},
	}

	for name := range healthMapping {
		key := types.NamespacedName{Namespace: ns, Name: name}
		status, err := resources.HealthyDeployment(ctx, r, key, healthMapping[name].minReady)
		if err != nil {
			return nil, fmt.Errorf("failed to get dep health %q: %w", name, err)
		}
		*healthMapping[name].healthStatus = kubermaticv1helper.GetHealthStatus(status, cluster, r.versions)
	}

	var err error
	key := types.NamespacedName{Namespace: ns, Name: resources.EtcdStatefulSetName}

	etcdHealthStatus, err := resources.HealthyStatefulSet(ctx, r, key, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd health: %w", err)
	}
	extendedHealth.Etcd = kubermaticv1helper.GetHealthStatus(etcdHealthStatus, cluster, r.versions)

	// check the actual status of the machineController components only if the API server is healthy
	// because we need to access it to retrieve the machineController mutatingWebhookConfiguration
	mcHealthStatus := kubermaticv1.HealthStatusDown
	if extendedHealth.Apiserver == kubermaticv1.HealthStatusUp {
		mcHealthStatus, err = r.machineControllerHealthCheck(ctx, cluster, ns)
		if err != nil {
			return nil, fmt.Errorf("failed to get machine controller health: %w", err)
		}
	}
	extendedHealth.MachineController = mcHealthStatus

	return extendedHealth, nil
}

func (r *Reconciler) syncHealth(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	extendedHealth, err := r.clusterHealth(ctx, cluster)
	if err != nil {
		return err
	}

	return kubermaticv1helper.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth = *extendedHealth

		// set ClusterConditionEtcdClusterInitialized, this should be done only once
		// when etcd becomes healthy for the first time.
		if extendedHealth.Etcd == kubermaticv1.HealthStatusUp {
			kubermaticv1helper.SetClusterCondition(
				c,
				r.versions,
				kubermaticv1.ClusterConditionEtcdClusterInitialized,
				corev1.ConditionTrue,
				"",
				"Etcd Cluster has been initialized successfully",
			)
		}

		if kubermaticv1helper.IsClusterInitialized(cluster, r.versions) {
			kubermaticv1helper.SetClusterCondition(
				c,
				r.versions,
				kubermaticv1.ClusterConditionClusterInitialized,
				corev1.ConditionTrue,
				"",
				"Cluster has been initialized successfully",
			)
		}
	})
}

func (r *Reconciler) machineControllerHealthCheck(ctx context.Context, cluster *kubermaticv1.Cluster, namespace string) (kubermaticv1.HealthStatus, error) {
	userClient, err := r.userClusterConnProvider.GetClient(ctx, cluster)
	if err != nil {
		return kubermaticv1.HealthStatusDown, err
	}

	// check the existence of the mutatingWebhookConfiguration in the user cluster
	key := types.NamespacedName{Name: resources.MachineControllerMutatingWebhookConfigurationName}
	webhookMutatingConf := &admissionregistrationv1.MutatingWebhookConfiguration{}
	err = userClient.Get(ctx, key, webhookMutatingConf)
	if err != nil && !kerrors.IsNotFound(err) {
		return kubermaticv1.HealthStatusDown, err
	}
	// if the mutatingWebhookConfiguration doesn't exist yet, return StatusDown
	if kerrors.IsNotFound(err) {
		return kubermaticv1.HealthStatusDown, nil
	}

	// check the machine controller deployment is healthy
	key = types.NamespacedName{Namespace: namespace, Name: resources.MachineControllerDeploymentName}
	mcStatus, err := resources.HealthyDeployment(ctx, r, key, 1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to get dep health %q: %w", resources.MachineControllerDeploymentName, err)
	}

	// check the machine controller webhook deployment is healthy
	key = types.NamespacedName{Namespace: namespace, Name: resources.MachineControllerWebhookDeploymentName}
	mcWebhookStatus, err := resources.HealthyDeployment(ctx, r, key, 1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to get dep health %q: %w", resources.MachineControllerWebhookDeploymentName, err)
	}

	switch {
	case mcStatus == kubermaticv1.HealthStatusUp && mcWebhookStatus == kubermaticv1.HealthStatusUp:
		return kubermaticv1.HealthStatusUp, nil
	case mcStatus == kubermaticv1.HealthStatusProvisioning || mcWebhookStatus == kubermaticv1.HealthStatusProvisioning:
		return kubermaticv1.HealthStatusProvisioning, nil
	default:
		return kubermaticv1.HealthStatusDown, nil
	}
}

func (r *Reconciler) statefulSetHealthCheck(ctx context.Context, c *kubermaticv1.Cluster) (bool, error) {
	// check the etcd
	statefulSet := &appsv1.StatefulSet{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.EtcdStatefulSetName}, statefulSet)

	if err != nil {
		// if the StatefulSet for etcd doesn't exist yet, there's nothing to worry about
		if kerrors.IsNotFound(err) {
			return true, nil
		} else {
			return false, err
		}
	}

	ready := statefulSet.Status.Replicas == statefulSet.Status.ReadyReplicas
	updated := statefulSet.Status.Replicas == statefulSet.Status.UpdatedReplicas

	return ready && updated, nil
}
