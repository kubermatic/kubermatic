/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package clusterdeletion

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupPolicyBindings(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	ns := cluster.Status.NamespaceName
	if ns == "" {
		return nil
	}

	// PolicyBindings live in the cluster namespace and their cleanup finalizer can
	// otherwise keep that namespace terminating. If the namespace is already gone,
	// the seed-side cleanup has nothing left to unblock.
	if err := d.seedClient.DeleteAllOf(ctx, &kubermaticv1.PolicyBinding{}, ctrlruntimeclient.InNamespace(ns)); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	bindings := &kubermaticv1.PolicyBindingList{}
	if err := d.seedClient.List(ctx, bindings, ctrlruntimeclient.InNamespace(ns)); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	for _, binding := range bindings.Items {
		if finalizer := kubermaticv1.PolicyBindingCleanupFinalizer; kuberneteshelper.HasFinalizer(&binding, finalizer) {
			log.Infow("Garbage-collecting PolicyBinding", "policyBinding", binding.Name)

			if err := kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, &binding, finalizer); err != nil {
				return fmt.Errorf("failed to remove PolicyBinding finalizer: %w", err)
			}
		}
	}

	return nil
}
