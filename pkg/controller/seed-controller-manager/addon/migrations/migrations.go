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

package migrations

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type AddonMigration interface {
	// Targets determines whether a given migration is applicable/relevant to a
	// given addon+cluster combination.
	Targets(cluster *kubermaticv1.Cluster, addonName string) bool
	// PreApply is called before an addon's rendered manifest is applied in a usercluster.
	// It's possible for this to be the initial installation, or any subsequent updates.
	// This function should ensure that the addon can be cleanly applied.
	PreApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error
	// PostApply is called right after an addon manifest was applied in a usercluster and
	// can be used to update the Cluster object with new status information or similar.
	PostApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error
	// PreRemove is called before an addon is removed from a usercluster.
	PreRemove(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error
	// PostRemove is called right after an addon was either removed (i.e. its manifest was
	// also already removed) or if an addon manifest renders into a empty string (e.g. the
	// csi addon, when CSIDrivers are disabled). This function should clean up what
	// `kubectl apply --prune` would not remove.
	PostRemove(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error
}

var allMigrations = []AddonMigration{
	&csiHetznerMigration{},
	&csiVsphereMigration{},
	&csiAzureRBACMigration{},
	&csiAzureHelmMigration{},
	&kubeStateMetricsMigration{},
}

func RelevantMigrations(cluster *kubermaticv1.Cluster, addonName string) AddonMigration {
	result := &aggregatedMigration{}

	for i, m := range allMigrations {
		if m.Targets(cluster, addonName) {
			result.children = append(result.children, allMigrations[i])
		}
	}

	return result
}

type nopMigration struct{}

var _ AddonMigration = &nopMigration{}

func (nopMigration) Targets(cluster *kubermaticv1.Cluster, addonName string) bool {
	panic("Implement this when embedding the nop migration in your code.")
}

func (nopMigration) PreApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	return nil
}

func (nopMigration) PostApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	return nil
}

func (nopMigration) PreRemove(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	return nil
}

func (nopMigration) PostRemove(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	return nil
}

type aggregatedMigration struct {
	children []AddonMigration
}

var _ AddonMigration = &aggregatedMigration{}

func (aggregatedMigration) Targets(cluster *kubermaticv1.Cluster, addonName string) bool {
	return true
}

func (m aggregatedMigration) PreApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	for _, child := range m.children {
		if err := child.PreApply(ctx, log, cluster, seedClient, userclusterClient); err != nil {
			return err
		}
	}

	return nil
}

func (m aggregatedMigration) PostApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	for _, child := range m.children {
		if err := child.PostApply(ctx, log, cluster, seedClient, userclusterClient); err != nil {
			return err
		}
	}

	return nil
}

func (m aggregatedMigration) PreRemove(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	for _, child := range m.children {
		if err := child.PreRemove(ctx, log, cluster, seedClient, userclusterClient); err != nil {
			return err
		}
	}

	return nil
}

func (m aggregatedMigration) PostRemove(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	for _, child := range m.children {
		if err := child.PostRemove(ctx, log, cluster, seedClient, userclusterClient); err != nil {
			return err
		}
	}

	return nil
}
