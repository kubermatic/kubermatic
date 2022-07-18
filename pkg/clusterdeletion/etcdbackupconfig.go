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

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupEtcdBackupConfigs(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, apiv1.EtcdBackupConfigCleanupFinalizer) {
		return nil
	}

	if cluster.Status.NamespaceName != "" {
		// always attempt to cleanup, even if the controllers might be disabled now
		backupConfigs := &kubermaticv1.EtcdBackupConfigList{}
		if err := d.seedClient.List(ctx, backupConfigs, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to get EtcdBackupConfigs: %w", err)
		}

		if len(backupConfigs.Items) > 0 {
			for _, backupConfig := range backupConfigs.Items {
				if err := d.seedClient.Delete(ctx, &backupConfig); err != nil {
					return fmt.Errorf("failed to delete EtcdBackupConfig %q: %w", backupConfig.Name, err)
				}
			}

			d.recorder.Eventf(cluster, corev1.EventTypeNormal, "EtcdBackupConfigCleanup", "There are %d EtcdBackupConfig objects waiting for deletion.", len(backupConfigs.Items))
			return nil
		}
	}

	d.recorder.Event(cluster, corev1.EventTypeNormal, "EtcdBackupConfigCleanup", "Cleanup has been completed.")

	return kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, cluster, apiv1.EtcdBackupConfigCleanupFinalizer)
}
