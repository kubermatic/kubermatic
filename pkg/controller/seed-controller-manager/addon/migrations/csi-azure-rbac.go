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

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Between v2.24 and v2.25, the roleRef in a ClusterRoleBinding for the Azure CSI changed.
type csiAzureRBACMigration struct {
	nopMigration
}

func (m *csiAzureRBACMigration) Targets(cluster *kubermaticv1.Cluster, addonName string) bool {
	return addonName == "csi" && cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] && cluster.Spec.Cloud.Azure != nil
}

func (m *csiAzureRBACMigration) PreApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	crb := &rbacv1.ClusterRoleBinding{}
	if err := userclusterClient.Get(ctx, types.NamespacedName{Name: "csi-azuredisk-node-secret-binding"}, crb); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get ClusterRoleBinding: %w", err)
	}

	if crb.RoleRef.Name != "csi-azuredisk-node-role" {
		log.Infof("Deleting Azure ClusterRoleBinding %s to allow upgrade", crb.Name)
		if err := userclusterClient.Delete(ctx, crb); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete ClusterRoleBinding: %w", err)
		}
	}

	return nil
}
