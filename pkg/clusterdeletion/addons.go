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

package clusterdeletion

import (
	"context"
	"fmt"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupAddons(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, apiv1.AddonCleanupFinalizer) {
		return nil
	}

	if cluster.Status.NamespaceName != "" {
		addons := kubermaticv1.AddonList{}
		if err := d.seedClient.List(ctx, &addons, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to list cluster addons: %w", err)
		}

		for _, addon := range addons.Items {
			if addon.DeletionTimestamp != nil {
				if err := d.seedClient.Delete(ctx, &addon); err != nil {
					return fmt.Errorf("failed to delete Addon %q: %w", addon.Name, err)
				}
			}
		}

		if len(addons.Items) > 0 {
			d.recorder.Eventf(cluster, corev1.EventTypeNormal, "AddonCleanup", "There are %d Addons waiting for deletion.", len(addons.Items))
			return nil
		}
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, cluster, apiv1.AddonCleanupFinalizer)
}
