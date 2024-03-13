/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package util

import (
	"context"
	"time"

	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DeployMatchesVersion returns true if the version label of the deployment matches the supplied version.
func DeployMatchesVersion(deploy *appsv1.Deployment, version string) bool {
	return deploy.Labels[resources.VersionLabel] == version
}

// DeployReplicasReady returns true if all replicas of a deployment are ready.
func DeployReplicasReady(deploy *appsv1.Deployment) bool {
	return deploy.Status.ReadyReplicas == *deploy.Spec.Replicas
}

// WaitForUpdatedDeployment queries k8s until a deployment with the supplied version exists or until the timeout is reached.
func WaitForUpdatedDeployment(ctx context.Context, deployToWatch *appsv1.Deployment, version string, kubeClient ctrlruntimeclient.Client, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 1*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		res := &appsv1.Deployment{}
		nsn := types.NamespacedName{Namespace: deployToWatch.Namespace, Name: deployToWatch.Name}

		if err := kubeClient.Get(ctx, nsn, res); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		versionCheck := DeployMatchesVersion(res, version)
		// we can exit early if the Version doesn't match, no need to check for replicas
		if !versionCheck {
			return false, nil
		}

		return DeployReplicasReady(res), nil
	})
}
