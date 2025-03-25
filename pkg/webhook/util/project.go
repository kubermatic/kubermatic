/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func OptimisticallyCheckIfProjectIsValid(ctx context.Context, client ctrlruntimeclient.Client, projectName string, isUpdate bool) error {
	if projectName == "" {
		return errors.New("project name must be configured")
	}

	project := &kubermaticv1.Project{}
	if err := client.Get(ctx, types.NamespacedName{Name: projectName}, project); err != nil {
		// We rely on eventual consistency; this webhook should only check if the UserSSHKey
		// object is consistent in itself.
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to get project: %w", err)
	}

	// Do not check the project phase, as projects only get Active after being successfully
	// reconciled. This requires the owner user to be setup properly as well, which in turn
	// requires owner references to be setup. All of this is super annoying when doing
	// GitOps. Instead we rely on _eventual_ consistency and only check that the project
	// exists and is not being deleted.
	if !isUpdate && project.DeletionTimestamp != nil {
		return errors.New("project is in deletion, cannot create new resources in it")
	}

	return nil
}
