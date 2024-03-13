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

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitForDeploymentRollout queries k8s until a deployment with the supplied version exists and
// has been rolled out, or until the timeout is reached.
func WaitForDeploymentRollout(ctx context.Context, client ctrlruntimeclient.Client, deployment *appsv1.Deployment, version string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 1*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		dep := &appsv1.Deployment{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(deployment), dep); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		if dep.Labels[resources.VersionLabel] != version {
			return false, nil
		}

		return kubernetes.IsDeploymentRolloutComplete(dep, 0)
	})
}
