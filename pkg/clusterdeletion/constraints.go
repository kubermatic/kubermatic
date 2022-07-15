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

package clusterdeletion

import (
	"context"
	"fmt"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupConstraints(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if ns := cluster.Status.NamespaceName; ns != "" {
		if err := d.seedClient.DeleteAllOf(ctx, &kubermaticv1.Constraint{}, ctrlruntimeclient.InNamespace(ns)); err != nil {
			return err
		}

		// `apiv1.GatekeeperConstraintCleanupFinalizer` is added by user-cluster-controller-manager/constraints-syncer.
		// It could be the case that during cluster deletion, user-cluster-controller-manager is deleted before it removes
		// the finalizer from constraints object, in this case, the user-cluster namespace will get stuck on deletion
		// (because the kubernetes controller in KKP will not re-create the user-cluster-controller-manager Deployment,
		// for example).
		// So here we just remove the finalizer from constraints so that user-cluster namespace can be garbage-collected.
		// Ref:https://github.com/kubermatic/kubermatic/issues/6934
		constraintList := &kubermaticv1.ConstraintList{}
		if err := d.seedClient.List(ctx, constraintList, ctrlruntimeclient.InNamespace(ns)); err != nil {
			return err
		}

		for _, constraint := range constraintList.Items {
			err := kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, &constraint, apiv1.GatekeeperConstraintCleanupFinalizer)
			if err != nil {
				return fmt.Errorf("failed to remove constraint finalizer %s: %w", constraint.Name, err)
			}
		}
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, cluster, apiv1.KubermaticConstraintCleanupFinalizer)
}
