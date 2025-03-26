//go:build ee

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package constraintsyncer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
)

func (r *reconciler) reconcile(ctx context.Context, constraint *kubermaticv1.Constraint, log *zap.SugaredLogger) error {
	finalizer := kubermaticv1.GatekeeperConstraintCleanupFinalizer

	// constraint deletion
	if constraint.Spec.Disabled {
		if err := r.cleanupConstraint(ctx, constraint, log); err != nil {
			return err
		}
	}

	if constraint.DeletionTimestamp != nil {
		if kuberneteshelper.HasFinalizer(constraint, finalizer) {
			if err := r.cleanupConstraint(ctx, constraint, log); err != nil {
				return err
			}
		}

		return kuberneteshelper.TryRemoveFinalizer(ctx, r.seedClient, constraint, finalizer)
	}

	if !constraint.Spec.Disabled {
		// add finalizer
		if err := kuberneteshelper.TryAddFinalizer(ctx, r.seedClient, constraint, finalizer); err != nil {
			return fmt.Errorf("failed to add finalizer: %w", err)
		}

		// constraint creation
		if err := r.createConstraint(ctx, constraint); err != nil {
			return err
		}
	}

	return nil
}
