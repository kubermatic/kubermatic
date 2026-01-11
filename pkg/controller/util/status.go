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

package util

import (
	"context"
	"reflect"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type AddonPatchFunc func(addon *kubermaticv1.Addon)

// UpdateAddonStatus will attempt to patch the status of the given addon.
func UpdateAddonStatus(ctx context.Context, client ctrlruntimeclient.Client, addon *kubermaticv1.Addon, patch AddonPatchFunc) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(addon)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the addon
		if err := client.Get(ctx, key, addon); err != nil {
			return err
		}

		// modify it
		original := addon.DeepCopy()
		patch(addon)

		// save some work
		if reflect.DeepEqual(original.Status, addon.Status) {
			return nil
		}

		// update the status
		return client.Status().Patch(ctx, addon, ctrlruntimeclient.MergeFrom(original))
	})
}

type ClusterPatchFunc func(cluster *kubermaticv1.Cluster)

// UpdateClusterStatus will attempt to patch the cluster status
// of the given cluster.
func UpdateClusterStatus(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, patch ClusterPatchFunc) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(cluster)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the cluster
		if err := client.Get(ctx, key, cluster); err != nil {
			return err
		}

		// modify it
		original := cluster.DeepCopy()
		patch(cluster)

		// save some work
		if reflect.DeepEqual(original.Status, cluster.Status) {
			return nil
		}

		// update the status
		return client.Status().Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original))
	})
}

type SeedPatchFunc func(seed *kubermaticv1.Seed)

func UpdateSeedStatus(ctx context.Context, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed, patch SeedPatchFunc) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(seed)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the seed
		if err := client.Get(ctx, key, seed); err != nil {
			return err
		}

		// modify it
		original := seed.DeepCopy()
		patch(seed)

		// save some work
		if reflect.DeepEqual(original.Status, seed.Status) {
			return nil
		}

		// update the status
		return client.Status().Patch(ctx, seed, ctrlruntimeclient.MergeFrom(original))
	})
}

type KubermaticConfigurationPatchFunc func(kc *kubermaticv1.KubermaticConfiguration)

func UpdateKubermaticConfigurationStatus(ctx context.Context,
	client ctrlruntimeclient.Client,
	kc *kubermaticv1.KubermaticConfiguration,
	patch KubermaticConfigurationPatchFunc,
) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(kc)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the Kubermatic Configuration
		if err := client.Get(ctx, key, kc); err != nil {
			return err
		}

		// modify it
		original := kc.DeepCopy()
		patch(kc)

		if reflect.DeepEqual(original.Status, kc.Status) {
			return nil
		}

		// update the status
		return client.Status().Patch(ctx, kc, ctrlruntimeclient.MergeFrom(original))
	})
}

type ResourceQuotaPatchFunc func(resourceQuota *kubermaticv1.ResourceQuota)

// UpdateResourceQuotaStatus will attempt to patch the resource quota status
// of the given resource quota.
func UpdateResourceQuotaStatus(ctx context.Context, client ctrlruntimeclient.Client, resourceQuota *kubermaticv1.ResourceQuota, patch ResourceQuotaPatchFunc) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(resourceQuota)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the resourceQuota
		if err := client.Get(ctx, key, resourceQuota); err != nil {
			return err
		}

		// modify it
		original := resourceQuota.DeepCopy()
		patch(resourceQuota)

		// save some work
		if reflect.DeepEqual(original.Status, resourceQuota.Status) {
			return nil
		}

		// update the status
		return client.Status().Patch(ctx, resourceQuota, ctrlruntimeclient.MergeFrom(original))
	})
}
