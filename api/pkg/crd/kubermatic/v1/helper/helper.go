package helper

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
