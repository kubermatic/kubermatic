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

package kubermaticmaster

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// migrateUserSSHKeyProjects takes care of setting spec.project (see #9421) on all existing SSH keys
// based on their owner references. This must happen before the new webhooks are installed.
// The keys are updated on the master cluster and are then synced downstream by the project
// controller. This syncing will fail as long as the seed clusters are not also updated (at least
// the CRDs), so in this sense it's safe to do it on the master.
// This function can be removed in KKP 2.22.
func (*MasterStack) migrateUserSSHKeyProjects(ctx context.Context, client ctrlruntimeclient.Client, logger logrus.FieldLogger, opt stack.DeployOptions) error {
	keys := &kubermaticv1.UserSSHKeyList{}
	if err := client.List(ctx, keys); err != nil {
		return fmt.Errorf("failed to list UserSSHKeys: %w", err)
	}

	apiVersion := kubermaticv1.SchemeGroupVersion.String()
	kind := kubermaticv1.ProjectKindName

	// determine if there are keys that need to be migrated
	needMigration := false
	for _, key := range keys.Items {
		if key.Spec.Project == "" {
			needMigration = true
			break
		}
	}

	// An old webhook could reject any changes to fields that it doesn't understand,
	// so we must remove the problematic webhooks prior to the migration; to ensure
	// the KKP operator doesn't bring them back, we have to scale it down first.
	// Once the migration is over and the regular installation/upgrade has finished,
	// the new operator will deploy a new webhook for us.
	if needMigration {
		logger.Info("Temporarily removing UserSSHKey webhook…")

		if err := util.ShutdownDeployment(ctx, logger, client, KubermaticOperatorNamespace, KubermaticOperatorDeploymentName); err != nil {
			return fmt.Errorf("failed to temporarily shut down the KKP operator: %w", err)
		}

		if err := util.WaitForAllPodsToBeGone(ctx, logger, client, KubermaticOperatorNamespace, KubermaticOperatorDeploymentName, 3*time.Minute); err != nil {
			return fmt.Errorf("failed to wait for KKP operator to shut down: %w", err)
		}

		if err := util.RemoveMutatingWebhook(ctx, logger, client, common.UserSSHKeyAdmissionWebhookName); err != nil {
			return fmt.Errorf("failed to remove the mutating webhook for UserSSHKeys: %w", err)
		}

		// Now that nobody should be blocking update operations on SSH keys, we can continue
		// with the migration.
		for _, key := range keys.Items {
			if key.Spec.Project != "" {
				continue
			}

			projectID := ""
			for _, ref := range key.OwnerReferences {
				if ref.APIVersion == apiVersion && ref.Kind == kind {
					if projectID != "" {
						return fmt.Errorf("key %s has multiple owner references to Projects, this should not be possible; reduce the owner refs to a single Project reference", key.Name)
					}

					projectID = ref.Name
				}
			}

			if projectID == "" {
				return fmt.Errorf("key %s no project owner reference, cannot determine project association", key.Name)
			}

			oldKey := key.DeepCopy()
			key.Spec.Project = projectID

			if err := client.Patch(ctx, &key, ctrlruntimeclient.MergeFrom(oldKey)); err != nil {
				return fmt.Errorf("failed to update key %s: %w", key.Name, err)
			}
		}
	}

	return nil
}

// migrateUserProjects takes care of setting spec.project (see #9441) on all existing service account users
// based on their owner references. This must happen before the new webhooks are installed.
// The users are updated on the master cluster and are then synced downstream by the project
// controller. This syncing will fail as long as the seed clusters are not also updated (at least
// the CRDs), so in this sense it's safe to do it on the master.
// This function can be removed in KKP 2.22.
func (*MasterStack) migrateUserProjects(ctx context.Context, client ctrlruntimeclient.Client, logger logrus.FieldLogger, opt stack.DeployOptions) error {
	users := &kubermaticv1.UserList{}
	if err := client.List(ctx, users); err != nil {
		return fmt.Errorf("failed to list Users: %w", err)
	}

	apiVersion := kubermaticv1.SchemeGroupVersion.String()
	kind := kubermaticv1.ProjectKindName

	for _, user := range users.Items {
		// already migrated
		if user.Spec.Project != "" {
			continue
		}

		if !kubermaticv1helper.IsProjectServiceAccount(user.Spec.Email) {
			continue
		}

		projectID := ""
		for _, ref := range user.OwnerReferences {
			if ref.APIVersion == apiVersion && ref.Kind == kind {
				if projectID != "" {
					return fmt.Errorf("user %s has multiple owner references to Projects, this should not be possible; reduce the owner refs to a single Project reference", user.Name)
				}

				projectID = ref.Name
			}
		}

		if projectID == "" {
			return fmt.Errorf("user %s has no project owner reference, cannot determine project association", user.Name)
		}

		oldUser := user.DeepCopy()
		user.Spec.Project = projectID

		if err := client.Patch(ctx, &user, ctrlruntimeclient.MergeFrom(oldUser)); err != nil {
			return fmt.Errorf("failed to update user %s: %w", user.Name, err)
		}
	}

	return nil
}

// migrateExternalClusterProviders takes care of setting the providername to BYO
// and adding the BYOCloudSpec to every ExternalCluster that is not using one of
// the other providers.
// This function can be removed in KKP 2.22.
func (*MasterStack) migrateExternalClusterProviders(ctx context.Context, client ctrlruntimeclient.Client, logger logrus.FieldLogger, opt stack.DeployOptions) error {
	clusters := &kubermaticv1.ExternalClusterList{}
	if err := client.List(ctx, clusters); err != nil {
		return fmt.Errorf("failed to list ExternalClusters: %w", err)
	}

	for _, cluster := range clusters.Items {
		if cluster.Spec.CloudSpec.ProviderName == "" {
			cluster.Spec.CloudSpec.ProviderName = kubermaticv1.ExternalClusterBringYourOwnProvider
			cluster.Spec.CloudSpec.BringYourOwn = &kubermaticv1.ExternalClusterBringYourOwnCloudSpec{}

			if err := client.Update(ctx, &cluster); err != nil {
				return fmt.Errorf("failed to update external cluster %s: %w", cluster.Name, err)
			}
		}
	}

	return nil
}
