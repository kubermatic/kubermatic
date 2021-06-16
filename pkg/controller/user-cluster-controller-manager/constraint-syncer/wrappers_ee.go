// +build ee

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

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *reconciler) reconcile(ctx context.Context, constraint *kubermaticv1.Constraint, log *zap.SugaredLogger) error {
	// constraint deletion
	if !constraint.Spec.Active {
		if err := r.cleanupConstraint(ctx, constraint, log); err != nil {
			return err
		}
	}

	if constraint.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(constraint, kubermaticapiv1.GatekeeperConstraintCleanupFinalizer) {
			return nil
		}

		if err := r.cleanupConstraint(ctx, constraint, log); err != nil {
			return err
		}

		oldConstraint := constraint.DeepCopy()
		kuberneteshelper.RemoveFinalizer(constraint, kubermaticapiv1.GatekeeperConstraintCleanupFinalizer)
		if err := r.seedClient.Patch(ctx, constraint, ctrlruntimeclient.MergeFrom(oldConstraint)); err != nil {
			return fmt.Errorf("failed to remove constraint finalizer %s: %v", constraint.Name, err)
		}
		return nil
	}

	if constraint.Spec.Active {
		// add finalizer
		if !kuberneteshelper.HasFinalizer(constraint, kubermaticapiv1.GatekeeperConstraintCleanupFinalizer) {
			oldConstraint := constraint.DeepCopy()
			kuberneteshelper.AddFinalizer(constraint, kubermaticapiv1.GatekeeperConstraintCleanupFinalizer)
			if err := r.seedClient.Patch(ctx, constraint, ctrlruntimeclient.MergeFrom(oldConstraint)); err != nil {
				return fmt.Errorf("failed to set constraint finalizer %s: %v", constraint.Name, err)
			}
		}
		// constraint creation
		if err := r.createConstraint(ctx, constraint, log); err != nil {
			return err
		}
	}

	return nil
}
