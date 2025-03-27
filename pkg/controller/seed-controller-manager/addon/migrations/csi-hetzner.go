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

package migrations

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Between v2.22 and v2.23, there was a change to hetzner CSI driver immutable field fsGroupPolicy
// as a result, the CSDriver resource has to be redeployed
// https://github.com/kubermatic/kubermatic/issues/12429
type csiHetznerMigration struct {
	nopMigration
}

func (m *csiHetznerMigration) Targets(cluster *kubermaticv1.Cluster, addonName string) bool {
	return addonName == "csi" && cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] && cluster.Spec.Cloud.Hetzner != nil
}

func (m *csiHetznerMigration) PreApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	driver := &storagev1.CSIDriver{}
	if err := userclusterClient.Get(ctx, types.NamespacedName{Name: "csi.hetzner.cloud"}, driver); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get CSIDriver: %w", err)
	}

	if driver.Spec.FSGroupPolicy == nil || *driver.Spec.FSGroupPolicy != storagev1.FileFSGroupPolicy {
		log.Info("Deleting Hetzner CSIDriver to allow upgrade")
		if err := userclusterClient.Delete(ctx, driver); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete old CSIDriver: %w", err)
		}
	}

	return nil
}
