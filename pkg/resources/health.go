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

package resources

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// HealthyDeployment tells if the deployment has a minimum of minReady replicas in Ready status
func HealthyDeployment(ctx context.Context, client ctrlruntimeclient.Client, nn types.NamespacedName, minReady int32) (kubermaticv1.HealthStatus, error) {
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
func HealthyStatefulSet(ctx context.Context, client ctrlruntimeclient.Client, nn types.NamespacedName, minReady int32) (kubermaticv1.HealthStatus, error) {
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
