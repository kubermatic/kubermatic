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

package helper

import (
	"context"
	"reflect"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ClusterPatchFunc func(cluster *kubermaticv1.Cluster)

// UpdateClusterStatus will attempt to patch the cluster status
// of the given cluster.
func UpdateClusterStatus(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, patch ClusterPatchFunc) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(cluster)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the cluster
		if err := client.Get(ctx, key, cluster); err != nil {
			return err
		}

		// modify it
		original := cluster.DeepCopy()
		patch(cluster)

		// save some work
		if reflect.DeepEqual(original.Status, cluster.Status) {
			return nil
		}

		// update the status
		return client.Status().Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original))
	})
}

// ClusterReconcileWrapper is a wrapper that should be used around
// any cluster reconciliaton. It:
//   - Checks if the cluster is paused
//   - Checks if the worker-name matches
//   - Sets the ReconcileSuccess condition for the controller by fetching
//     the current Cluster object and patching its status.
func ClusterReconcileWrapper(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	workerName string,
	cluster *kubermaticv1.Cluster,
	versions kubermatic.Versions,
	conditionType kubermaticv1.ClusterConditionType,
	reconcile func() (*reconcile.Result, error),
) (*reconcile.Result, error) {
	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != workerName {
		return nil, nil
	}
	if cluster.Spec.Pause {
		return nil, nil
	}

	reconcilingStatus := corev1.ConditionFalse
	result, err := reconcile()

	// Only set to true if we had no error and don't want to reqeue the cluster
	if err == nil && (result == nil || (!result.Requeue && result.RequeueAfter == 0)) {
		reconcilingStatus = corev1.ConditionTrue
	}

	errs := []error{err}
	if conditionType != kubermaticv1.ClusterConditionNone {
		err = UpdateClusterStatus(ctx, client, cluster, func(c *kubermaticv1.Cluster) {
			SetClusterCondition(c, versions, conditionType, reconcilingStatus, "", "")
		})
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			errs = append(errs, err)
		}
	}

	return result, utilerrors.NewAggregate(errs)
}

// SetClusterCondition sets a condition on the given cluster using the provided type, status,
// reason and message. It also adds the Kubermatic version and timestamps.
func SetClusterCondition(
	c *kubermaticv1.Cluster,
	v kubermatic.Versions,
	conditionType kubermaticv1.ClusterConditionType,
	status corev1.ConditionStatus,
	reason string,
	message string,
) {
	newCondition := kubermaticv1.ClusterCondition{
		Status:            status,
		KubermaticVersion: uniqueVersion(v),
		Reason:            reason,
		Message:           message,
	}

	oldCondition, hadCondition := c.Status.Conditions[conditionType]
	if hadCondition {
		conditionCopy := oldCondition.DeepCopy()

		// Reset the times before comparing
		conditionCopy.LastHeartbeatTime.Reset()
		conditionCopy.LastTransitionTime.Reset()

		if apiequality.Semantic.DeepEqual(*conditionCopy, newCondition) {
			return
		}
	}

	now := metav1.Now()
	newCondition.LastHeartbeatTime = now
	newCondition.LastTransitionTime = oldCondition.LastTransitionTime
	if hadCondition && oldCondition.Status != status {
		newCondition.LastTransitionTime = now
	}

	if c.Status.Conditions == nil {
		c.Status.Conditions = map[kubermaticv1.ClusterConditionType]kubermaticv1.ClusterCondition{}
	}
	c.Status.Conditions[conditionType] = newCondition
}

// ClusterReconciliationSuccessful checks if cluster has all conditions that are
// required for it to be healthy. ignoreKubermaticVersion should only be set in tests.
func ClusterReconciliationSuccessful(cluster *kubermaticv1.Cluster, versions kubermatic.Versions, ignoreKubermaticVersion bool) (missingConditions []kubermaticv1.ClusterConditionType, success bool) {
	conditionsToExclude := []kubermaticv1.ClusterConditionType{kubermaticv1.ClusterConditionSeedResourcesUpToDate}
	for _, conditionType := range kubermaticv1.AllClusterConditionTypes {
		if conditionTypeListHasConditionType(conditionsToExclude, conditionType) {
			continue
		}

		if !clusterHasCurrentSuccessfullConditionType(cluster, versions, conditionType, ignoreKubermaticVersion) {
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

func clusterHasCurrentSuccessfullConditionType(
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
	isInitialized := cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionClusterInitialized, corev1.ConditionTrue)
	// If was set to true at least once just return true
	if isInitialized {
		return true
	}

	_, success := ClusterReconciliationSuccessful(cluster, versions, false)
	upToDate := cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionSeedResourcesUpToDate, corev1.ConditionTrue)
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
	return cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted, corev1.ConditionTrue)
}
