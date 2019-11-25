package helper

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ClusterReconcileWrapper is a wrapper that should be used around
// any cluster reconciliaton. It:
// * Checks if the cluster is paused
// * Checks if the worker-name matches
// * Sets the ReconcileSuccess condition for the controller
func ClusterReconcileWrapper(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	workerName string,
	cluster *kubermaticv1.Cluster,
	conditionType kubermaticv1.ClusterConditionType,
	reconcile func() (*reconcile.Result, error)) (*reconcile.Result, error) {

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
	oldCluster := cluster.DeepCopy()
	SetClusterCondition(cluster, conditionType, reconcilingStatus, "", "")
	errs = append(errs, ctrlruntimeclient.IgnoreNotFound(client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))))
	return result, utilerrors.NewAggregate(errs)
}

// GetClusterCondition returns the index of the given condition or -1 and the condition itself
// or a nilpointer.
func GetClusterCondition(c *kubermaticv1.Cluster, conditionType kubermaticv1.ClusterConditionType) (int, *kubermaticv1.ClusterCondition) {
	for i, condition := range c.Status.Conditions {
		if conditionType == condition.Type {
			return i, &condition
		}
	}
	return -1, nil
}

// SetClusterCondition sets a condition on the given cluster using the provided type, status,
// reason and message. It also adds the Kubermatic version and tiemstamps.
func SetClusterCondition(
	c *kubermaticv1.Cluster,
	conditionType kubermaticv1.ClusterConditionType,
	status corev1.ConditionStatus,
	reason string,
	message string,
) {
	newCondition := kubermaticv1.ClusterCondition{
		Type:              conditionType,
		Status:            status,
		KubermaticVersion: resources.KUBERMATICGITTAG + "-" + resources.KUBERMATICCOMMIT,
		Reason:            reason,
		Message:           message,
	}
	pos, oldCondition := GetClusterCondition(c, conditionType)
	if oldCondition != nil {
		// Reset the times before comparing
		oldCondition.LastHeartbeatTime.Reset()
		oldCondition.LastTransitionTime.Reset()
		if apiequality.Semantic.DeepEqual(*oldCondition, newCondition) {
			return
		}
	}

	newCondition.LastHeartbeatTime = metav1.Now()
	if oldCondition != nil && oldCondition.Status != status {
		newCondition.LastTransitionTime = metav1.Now()
	}

	if oldCondition != nil {
		c.Status.Conditions[pos] = newCondition
	} else {
		c.Status.Conditions = append(c.Status.Conditions, newCondition)
	}
}

func ClusterReconciliationSuccessful(cluster *kubermaticv1.Cluster) (missingConditions []kubermaticv1.ClusterConditionType, success bool) {
	conditionsToExclude := []kubermaticv1.ClusterConditionType{kubermaticv1.ClusterConditionSeedResourcesUpToDate}
	if cluster.IsOpenshift() {
		conditionsToExclude = append(conditionsToExclude, kubermaticv1.ClusterConditionClusterControllerReconcilingSuccess)
	}
	if cluster.IsKubernetes() {
		conditionsToExclude = append(conditionsToExclude, kubermaticv1.ClusterConditionOpenshiftControllerReconcilingSuccess)
	}

	for _, conditionType := range kubermaticv1.AllClusterConditionTypes {
		if conditionTypeListHasConditionType(conditionsToExclude, conditionType) {
			continue
		}

		if !clusterHasCurrentSuccessfullConditionType(cluster, conditionType) {
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
	conditionType kubermaticv1.ClusterConditionType,
) bool {
	for _, condition := range cluster.Status.Conditions {
		if condition.Type != conditionType {
			continue
		}

		if condition.Status != corev1.ConditionTrue {
			return false
		}

		if condition.KubermaticVersion != resources.KUBERMATICGITTAG+"-"+resources.KUBERMATICCOMMIT {
			return false
		}

		return true
	}

	return false
}

func IsClusterInitialized(cluster *kubermaticv1.Cluster) bool {
	isInitialized := cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionClusterInitialized, corev1.ConditionTrue)
	// If was set to true at least once just return true
	if isInitialized {
		return true
	}

	_, success := ClusterReconciliationSuccessful(cluster)
	upToDate := cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionSeedResourcesUpToDate, corev1.ConditionTrue)
	return success && upToDate && cluster.Status.ExtendedHealth.AllHealthy()
}

// We assume that te cluster is still provisioning if it was not initialized fully at least once.
func GetHealthStatus(status kubermaticv1.HealthStatus, cluster *kubermaticv1.Cluster) kubermaticv1.HealthStatus {
	if status == kubermaticv1.HealthStatusDown && !IsClusterInitialized(cluster) {
		return kubermaticv1.HealthStatusProvisioning
	}

	return status
}
