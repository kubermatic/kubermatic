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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/applications"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	kyvernocommonseedresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/common"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		resources.UserClusterControllerDeploymentName: {healthStatus: &extendedHealth.UserClusterControllerManager, minReady: 1},
	}

	showKonnectivity := cluster.Spec.ClusterNetwork.KonnectivityEnabled != nil && *cluster.Spec.ClusterNetwork.KonnectivityEnabled //nolint:staticcheck
	for name := range healthMapping {
		key := types.NamespacedName{Namespace: ns, Name: name}
		status, err := resources.HealthyDeployment(ctx, r, key, healthMapping[name].minReady)
		if err != nil {
			return nil, fmt.Errorf("failed to determine deployment's health %q: %w", name, err)
		}
		if healthMapping[name].healthStatus == nil {
			healthMapping[name].healthStatus = new(kubermaticv1.HealthStatus)
		}
		*healthMapping[name].healthStatus = util.GetHealthStatus(status, cluster, r.versions)
	}

	if showKonnectivity {
		// because konnectivity server is in apiserver pod
		extendedHealth.Konnectivity = extendedHealth.Apiserver
	}

	var err error
	key := types.NamespacedName{Namespace: ns, Name: resources.EtcdStatefulSetName}

	etcdHealthStatus, err := resources.HealthyStatefulSet(ctx, r, key, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd health: %w", err)
	}
	extendedHealth.Etcd = util.GetHealthStatus(etcdHealthStatus, cluster, r.versions)

	// MachineController is not deployed/supported on Edge clusters.
	if cluster.Spec.Cloud.Edge == nil {
		// check the actual status of the machineController components only if the API server is healthy
		// because we need to access it to retrieve the machineController mutatingWebhookConfiguration
		mcHealthStatus := kubermaticv1.HealthStatusDown
		if extendedHealth.Apiserver == kubermaticv1.HealthStatusUp {
			mcHealthStatus, err = r.machineControllerHealthCheck(ctx, cluster, ns)
			if err != nil {
				return nil, fmt.Errorf("failed to get machine controller health: %w", err)
			}
		}
		extendedHealth.MachineController = util.GetHealthStatus(mcHealthStatus, cluster, r.versions)
	}

	applicationControllerHealthStatus := kubermaticv1.HealthStatusDown
	if extendedHealth.Apiserver == kubermaticv1.HealthStatusUp {
		applicationControllerHealthStatus, err = r.applicationControllerHealthCheck(ctx, cluster, ns)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate application controller health: %w", err)
		}
	}
	extendedHealth.ApplicationController = util.GetHealthStatus(applicationControllerHealthStatus, cluster, r.versions)

	status, err := r.operatingSystemManagerHealthCheck(ctx, cluster, ns)
	if err != nil {
		return nil, fmt.Errorf("failed to get operating-system-manager health: %w", err)
	}
	extendedHealth.OperatingSystemManager = &status

	if cluster.Spec.IsKubernetesDashboardEnabled() {
		status, err := r.kubernetesDashboardHealthCheck(ctx, cluster, ns)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes-dashboard health: %w", err)
		}
		extendedHealth.KubernetesDashboard = &status
	}

	if cluster.Spec.IsKubeLBEnabled() {
		status, err := r.kubeLBHealthCheck(ctx, cluster, ns)
		if err != nil {
			return nil, fmt.Errorf("failed to get KubeLB health: %w", err)
		}
		extendedHealth.KubeLB = &status
	}

	if cluster.Spec.IsKyvernoEnabled() {
		status, err := r.kyvernoHealthCheck(ctx, cluster, ns)
		if err != nil {
			return nil, fmt.Errorf("failed to get Kyverno health: %w", err)
		}
		extendedHealth.Kyverno = &status
	}

	return extendedHealth, nil
}

func (r *Reconciler) syncHealth(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	extendedHealth, err := r.clusterHealth(ctx, cluster)
	if err != nil {
		return err
	}

	return util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth = *extendedHealth

		// set ClusterConditionEtcdClusterInitialized, this should be done only once
		// when etcd becomes healthy for the first time.
		if extendedHealth.Etcd == kubermaticv1.HealthStatusUp {
			util.SetClusterCondition(
				c,
				r.versions,
				kubermaticv1.ClusterConditionEtcdClusterInitialized,
				corev1.ConditionTrue,
				"",
				"Etcd Cluster has been initialized successfully",
			)
		}

		if util.IsClusterInitialized(cluster, r.versions) {
			util.SetClusterCondition(
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
	if err != nil && !apierrors.IsNotFound(err) {
		return kubermaticv1.HealthStatusDown, err
	}
	// if the mutatingWebhookConfiguration doesn't exist yet, return StatusDown
	if apierrors.IsNotFound(err) {
		return kubermaticv1.HealthStatusDown, nil
	}

	// check the machine controller deployment is healthy
	key = types.NamespacedName{Namespace: namespace, Name: resources.MachineControllerDeploymentName}
	mcStatus, err := resources.HealthyDeployment(ctx, r, key, 1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to determine deployment's health %q: %w", resources.MachineControllerDeploymentName, err)
	}

	// check the machine controller webhook deployment is healthy
	key = types.NamespacedName{Namespace: namespace, Name: resources.MachineControllerWebhookDeploymentName}
	mcWebhookStatus, err := resources.HealthyDeployment(ctx, r, key, 1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to determine deployment's health %q: %w", resources.MachineControllerWebhookDeploymentName, err)
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

func (r *Reconciler) operatingSystemManagerHealthCheck(ctx context.Context, cluster *kubermaticv1.Cluster, namespace string) (kubermaticv1.HealthStatus, error) {
	// check for the health of operating-system-manager deployment.
	key := types.NamespacedName{Namespace: namespace, Name: resources.OperatingSystemManagerDeploymentName}
	status, err := resources.HealthyDeployment(ctx, r, key, 1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to determine deployment's health %q: %w", resources.OperatingSystemManagerDeploymentName, err)
	}
	status = util.GetHealthStatus(status, cluster, r.versions)
	return status, nil
}

func (r *Reconciler) kubeLBHealthCheck(ctx context.Context, cluster *kubermaticv1.Cluster, namespace string) (kubermaticv1.HealthStatus, error) {
	// check for the health of kubeLB deployment.
	key := types.NamespacedName{Namespace: namespace, Name: resources.KubeLBDeploymentName}
	status, err := resources.HealthyDeployment(ctx, r, key, 1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to determine deployment's health %q: %w", resources.KubeLBDeploymentName, err)
	}
	status = util.GetHealthStatus(status, cluster, r.versions)
	return status, nil
}

func (r *Reconciler) kubernetesDashboardHealthCheck(ctx context.Context, cluster *kubermaticv1.Cluster, namespace string) (kubermaticv1.HealthStatus, error) {
	// check for the health of kubernetes-dashboard deployment.
	key := types.NamespacedName{Namespace: namespace, Name: resources.KubernetesDashboardDeploymentName}
	status, err := resources.HealthyDeployment(ctx, r, key, 1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to determine deployment's health %q: %w", resources.KubernetesDashboardDeploymentName, err)
	}
	status = util.GetHealthStatus(status, cluster, r.versions)
	return status, nil
}

// applicationControllerHealthCheck will check the health of all components that are required for Application controller to work
// We have intentionally created a dedicated health check for this as the resources are scattered through different components and the list of these required
// resources will grow with time.
func (r *Reconciler) applicationControllerHealthCheck(ctx context.Context, cluster *kubermaticv1.Cluster, namespace string) (kubermaticv1.HealthStatus, error) {
	userClient, err := r.userClusterConnProvider.GetClient(ctx, cluster)
	if err != nil {
		return kubermaticv1.HealthStatusDown, err
	}

	// Ensure that the ValidatingWebhookConfiguration for ApplicationInstallations exists in the user cluster
	// Invalid resources can leak through if this resource doesn't exist
	key := types.NamespacedName{Name: applications.ApplicationInstallationAdmissionWebhookName}
	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	err = userClient.Get(ctx, key, webhook)
	if err != nil && !apierrors.IsNotFound(err) {
		return kubermaticv1.HealthStatusDown, err
	}
	// if the ValidatingWebhookConfiguration doesn't exist yet, return StatusDown
	if apierrors.IsNotFound(err) {
		return kubermaticv1.HealthStatusDown, nil
	}

	// Ensure that the user-cluster-controller is healthy
	// application-installation-controller is part of the usercluster-controller manager
	key = types.NamespacedName{Namespace: namespace, Name: resources.UserClusterControllerDeploymentName}
	userClusterControllerStatus, err := resources.HealthyDeployment(ctx, r, key, 1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to determine deployment's health %q: %w", resources.UserClusterControllerDeploymentName, err)
	}

	// Ensure that the deployment for user-cluster-webhook is healthy
	key = types.NamespacedName{Namespace: namespace, Name: resources.UserClusterWebhookDeploymentName}
	userClusterWebhookStatus, err := resources.HealthyDeployment(ctx, r, key, 1)
	if err != nil {
		return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to determine deployment's health %q: %w", resources.UserClusterWebhookDeploymentName, err)
	}

	switch {
	case userClusterControllerStatus == kubermaticv1.HealthStatusUp && userClusterWebhookStatus == kubermaticv1.HealthStatusUp:
		return kubermaticv1.HealthStatusUp, nil
	case userClusterControllerStatus == kubermaticv1.HealthStatusProvisioning || userClusterWebhookStatus == kubermaticv1.HealthStatusProvisioning:
		return kubermaticv1.HealthStatusProvisioning, nil
	default:
		return kubermaticv1.HealthStatusDown, nil
	}
}

func (r *Reconciler) statefulSetHealthCheck(ctx context.Context, c *kubermaticv1.Cluster) (bool, error) {
	// check the etcd
	statefulSet := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.EtcdStatefulSetName}, statefulSet)

	if err != nil {
		// if the StatefulSet for etcd doesn't exist yet, there's nothing to worry about
		if apierrors.IsNotFound(err) {
			return true, nil
		} else {
			return false, err
		}
	}

	ready := statefulSet.Status.Replicas == statefulSet.Status.ReadyReplicas
	updated := statefulSet.Status.Replicas == statefulSet.Status.UpdatedReplicas

	return ready && updated, nil
}

func (r *Reconciler) kyvernoHealthCheck(ctx context.Context, cluster *kubermaticv1.Cluster, namespace string) (kubermaticv1.HealthStatus, error) {
	// Kyverno consists of 4 deployments with different replica counts
	type kyvernoDeployment struct {
		name     string
		minReady int32
	}

	kyvernoDeployments := []kyvernoDeployment{
		{
			name:     kyvernocommonseedresources.KyvernoAdmissionControllerDeploymentName,
			minReady: kyvernocommonseedresources.KyvernoAdmissionControllerReplicas,
		},
		{
			name:     kyvernocommonseedresources.KyvernoBackgroundControllerDeploymentName,
			minReady: kyvernocommonseedresources.KyvernoBackgroundControllerReplicas,
		},
		{
			name:     kyvernocommonseedresources.KyvernoCleanupControllerDeploymentName,
			minReady: kyvernocommonseedresources.KyvernoCleanupControllerReplicas,
		},
		{
			name:     kyvernocommonseedresources.KyvernoReportsControllerDeploymentName,
			minReady: kyvernocommonseedresources.KyvernoReportsControllerReplicas,
		},
	}

	allUp := true
	anyProvisioning := false

	for _, deployment := range kyvernoDeployments {
		key := types.NamespacedName{Namespace: namespace, Name: deployment.name}
		status, err := resources.HealthyDeployment(ctx, r, key, deployment.minReady)
		if err != nil {
			return kubermaticv1.HealthStatusDown, fmt.Errorf("failed to determine deployment's health %q: %w", deployment.name, err)
		}

		switch status {
		case kubermaticv1.HealthStatusDown:
			return kubermaticv1.HealthStatusDown, nil
		case kubermaticv1.HealthStatusProvisioning:
			anyProvisioning = true
			allUp = false
		case kubermaticv1.HealthStatusUp:
		}
	}

	switch {
	case allUp:
		status := util.GetHealthStatus(kubermaticv1.HealthStatusUp, cluster, r.versions)
		return status, nil
	case anyProvisioning:
		status := util.GetHealthStatus(kubermaticv1.HealthStatusProvisioning, cluster, r.versions)
		return status, nil
	default:
		status := util.GetHealthStatus(kubermaticv1.HealthStatusDown, cluster, r.versions)
		return status, nil
	}
}
