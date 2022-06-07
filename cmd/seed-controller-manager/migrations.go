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

package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// migrateClusterAddresses copies the `address` data from a Cluster into `status.address`.
// It leaves the old data behind, so that the migration is safe and even if an old controller
// tries to reconcile, it still sees valid data.
// In KKP 2.22 we can remove the `address` field entirely.
func migrateClusterAddresses(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client) error {
	log.Info("Migrating Cluster addresses")

	clusters := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, clusters); err != nil {
		return fmt.Errorf("failed to list Clusters: %w", err)
	}

	for _, cluster := range clusters.Items {
		oldCluster := cluster.DeepCopy()
		newAddress := &cluster.Status.Address
		oldAddress := cluster.Address

		if newAddress.AdminToken == "" {
			newAddress.AdminToken = oldAddress.AdminToken
		}

		if newAddress.ExternalName == "" {
			newAddress.ExternalName = oldAddress.ExternalName
		}

		if newAddress.InternalName == "" {
			newAddress.InternalName = oldAddress.InternalName
		}

		if newAddress.URL == "" {
			newAddress.URL = oldAddress.URL
		}

		if newAddress.IP == "" {
			newAddress.IP = oldAddress.IP
		}

		if newAddress.Port == 0 {
			newAddress.Port = oldAddress.Port
		}

		if err := client.Status().Patch(ctx, &cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return fmt.Errorf("failed to update cluster %s: %w", cluster.Name, err)
		}
	}

	return nil
}
