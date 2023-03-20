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

package cluster

import (
	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
)

var RequiredHealthConditions = []kubermaticv1.ClusterConditionType{
	kubermaticv1.ClusterConditionAddonControllerReconcilingSuccess,
	kubermaticv1.ClusterConditionBackupControllerReconcilingSuccess,
	kubermaticv1.ClusterConditionCloudControllerReconcilingSuccess,
	kubermaticv1.ClusterConditionClusterControllerReconcilingSuccess,
	kubermaticv1.ClusterConditionMonitoringControllerReconcilingSuccess,
	kubermaticv1.ClusterConditionSeedResourcesUpToDate,
	kubermaticv1.ClusterConditionUpdateControllerReconcilingSuccess,
}

// ClusterReconciliationSuccessful checks if cluster has all conditions that are
// required for it to be healthy. ignoreKubermaticVersion should only be set in tests.
func ClusterReconciliationSuccessful(cluster *kubermaticv1.Cluster, versions kubermatic.Versions, ignoreKubermaticVersion bool) (missingConditions []kubermaticv1.ClusterConditionType, success bool) {
	conditionsToExclude := []kubermaticv1.ClusterConditionType{kubermaticv1.ClusterConditionSeedResourcesUpToDate}
	for _, conditionType := range RequiredHealthConditions {
		if conditionTypeListHasConditionType(conditionsToExclude, conditionType) {
			continue
		}

		if !clusterHasCurrentSuccessfulConditionType(cluster, versions, conditionType, ignoreKubermaticVersion) {
			missingConditions = append(missingConditions, conditionType)
		}
	}

	return missingConditions, len(missingConditions) == 0
}

func conditionTypeListHasConditionType(
	list []kubermaticv1.ClusterConditionType,
	t kubermaticv1.ClusterConditionType,
) bool {
	for _, item := range list {
		if item == t {
			return true
		}
	}
	return false
}

func clusterHasCurrentSuccessfulConditionType(
	cluster *kubermaticv1.Cluster,
	versions kubermatic.Versions,
	conditionType kubermaticv1.ClusterConditionType,
	ignoreKubermaticVersion bool,
) bool {
	condition, exists := cluster.Status.Conditions[conditionType]
	if !exists {
		return false
	}

	if condition.Status != corev1.ConditionTrue {
		return false
	}

	if !ignoreKubermaticVersion && (condition.KubermaticVersion != uniqueVersion(versions)) {
		return false
	}

	return true
}

func IsClusterInitialized(cluster *kubermaticv1.Cluster, versions kubermatic.Versions) bool {
	isInitialized := cluster.Status.Conditions[kubermaticv1.ClusterConditionClusterInitialized].Status == corev1.ConditionTrue
	// If was set to true at least once just return true
	if isInitialized {
		return true
	}

	_, success := ClusterReconciliationSuccessful(cluster, versions, false)
	upToDate := cluster.Status.Conditions[kubermaticv1.ClusterConditionSeedResourcesUpToDate].Status == corev1.ConditionTrue
	return success && upToDate && cluster.Status.ExtendedHealth.AllHealthy()
}

// We assume that the cluster is still provisioning if it was not initialized fully at least once.
func GetHealthStatus(status kubermaticv1.HealthStatus, cluster *kubermaticv1.Cluster, versions kubermatic.Versions) kubermaticv1.HealthStatus {
	if status == kubermaticv1.HealthStatusDown && !IsClusterInitialized(cluster, versions) {
		return kubermaticv1.HealthStatusProvisioning
	}

	return status
}

func uniqueVersion(v kubermatic.Versions) string {
	return v.KubermaticCommit
}

func NeedCCMMigration(cluster *kubermaticv1.Cluster) bool {
	_, ccmOk := cluster.Annotations[kubermaticv1.CCMMigrationNeededAnnotation]
	_, csiOk := cluster.Annotations[kubermaticv1.CSIMigrationNeededAnnotation]

	return ccmOk && csiOk && !CCMMigrationCompleted(cluster)
}

func CCMMigrationCompleted(cluster *kubermaticv1.Cluster) bool {
	return cluster.Status.Conditions[kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted].Status == corev1.ConditionTrue
}
