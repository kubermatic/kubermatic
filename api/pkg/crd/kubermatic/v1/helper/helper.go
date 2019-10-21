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

// ClusterConditionSettingReconcileWrapper is a wrapper around
// a controllers reconcile that sets the ReconcileSuccess condition
// for that controller.
func ClusterConditionSettingReconcileWrapper(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	clusterName string,
	conditionType kubermaticv1.ClusterConditionType,
	reconcile func() (*reconcile.Result, error)) (*reconcile.Result, error) {

	reconcilingStatus := corev1.ConditionFalse
	result, err := reconcile()
	// Only set to true if we are completely done with this cluster
	if result != nil && !result.Requeue && result.RequeueAfter != 0 && err == nil {
		reconcilingStatus = corev1.ConditionTrue
	}
	errs := []error{err}
	errs = append(errs, clusterUpdater(ctx, client, clusterName, func(c *kubermaticv1.Cluster) {
		SetClusterCondition(c, conditionType, reconcilingStatus, "", "")
	}))
	return result, utilerrors.NewAggregate(errs)
}

func clusterUpdater(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	name string,
	modify func(*kubermaticv1.Cluster),
) error {
	return nil
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
