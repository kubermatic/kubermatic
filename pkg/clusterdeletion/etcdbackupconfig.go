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

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupEtcdBackupConfigs(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.EtcdBackupConfigCleanupFinalizer) {
		return nil
	}

	backupConfigs := &kubermaticv1.EtcdBackupConfigList{}
	if err := d.seedClient.List(ctx, backupConfigs, controllerruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
		return fmt.Errorf("failed to get EtcdBackupConfigs: %v", err)
	}

	if len(backupConfigs.Items) > 0 {
		for _, backupConfig := range backupConfigs.Items {
			if err := d.seedClient.Delete(ctx, &backupConfig); err != nil {
				return fmt.Errorf("failed to delete EtcdBackupConfig %q: %v", backupConfig.Name, err)
			}
		}
		return nil
	}

	oldCluster := cluster.DeepCopy()
	kuberneteshelper.RemoveFinalizer(cluster, kubermaticapiv1.EtcdBackupConfigCleanupFinalizer)
	return d.seedClient.Patch(ctx, cluster, controllerruntimeclient.MergeFrom(oldCluster))
}
