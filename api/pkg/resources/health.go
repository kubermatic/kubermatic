package resources

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HealthyDeployment tells if the deployment has a minimum of minReady replicas in Ready status
func HealthyDeployment(ctx context.Context, client client.Client, nn types.NamespacedName, minReady int32) (bool, error) {
	deployment := &appsv1.Deployment{}
	if err := client.Get(ctx, nn, deployment); err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return deployment.Status.ReadyReplicas >= minReady, nil
}

// HealthyStatefulSe tells if the deployment has a minimum of minReady replicas in Ready status
func HealthyStatefulSet(ctx context.Context, client client.Client, nn types.NamespacedName, minReady int32) (bool, error) {
	statefulSet := &appsv1.StatefulSet{}
	if err := client.Get(ctx, nn, statefulSet); err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return statefulSet.Status.ReadyReplicas >= minReady, nil
}
