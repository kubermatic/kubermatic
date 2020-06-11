package resources

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HealthyDeployment tells if the deployment has a minimum of minReady replicas in Ready status
func HealthyDeployment(ctx context.Context, client client.Client, nn types.NamespacedName, minReady int32) (kubermaticv1.HealthStatus, error) {
	deployment := &appsv1.Deployment{}
	if err := client.Get(ctx, nn, deployment); err != nil {
		if kerrors.IsNotFound(err) {
			return kubermaticv1.HealthStatusDown, nil
		}
		return kubermaticv1.HealthStatusDown, err
	}

	if deployment.Status.ReadyReplicas < minReady {
		return kubermaticv1.HealthStatusDown, nil
	}
	// update scenario
	if deployment.Status.UpdatedReplicas != *deployment.Spec.Replicas || deployment.Status.ReadyReplicas != *deployment.Spec.Replicas || deployment.Status.Replicas != *deployment.Spec.Replicas {
		return kubermaticv1.HealthStatusProvisioning, nil
	}
	return kubermaticv1.HealthStatusUp, nil
}

// HealthyStatefulSe tells if the deployment has a minimum of minReady replicas in Ready status
func HealthyStatefulSet(ctx context.Context, client client.Client, nn types.NamespacedName, minReady int32) (kubermaticv1.HealthStatus, error) {
	statefulSet := &appsv1.StatefulSet{}
	if err := client.Get(ctx, nn, statefulSet); err != nil {
		if kerrors.IsNotFound(err) {
			return kubermaticv1.HealthStatusDown, nil
		}
		return kubermaticv1.HealthStatusDown, err
	}

	if statefulSet.Status.ReadyReplicas < minReady {
		return kubermaticv1.HealthStatusDown, nil
	}
	if statefulSet.Status.UpdatedReplicas != *statefulSet.Spec.Replicas || statefulSet.Status.ReadyReplicas != *statefulSet.Spec.Replicas || statefulSet.Status.Replicas != *statefulSet.Spec.Replicas {
		return kubermaticv1.HealthStatusProvisioning, nil
	}
	return kubermaticv1.HealthStatusUp, nil
}
