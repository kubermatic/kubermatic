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

package kubernetes

import (
	"context"
	"reflect"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"

	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

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

type DatacenterPatchFunc func(datacenter *kubermaticv1.Datacenter)

func UpdateDatacenterStatus(ctx context.Context, client ctrlruntimeclient.Client, datacenter *kubermaticv1.Datacenter, patch DatacenterPatchFunc) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(datacenter)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the datacenter
		if err := client.Get(ctx, key, datacenter); err != nil {
			return err
		}

		// modify it
		original := datacenter.DeepCopy()
		patch(datacenter)

		// save some work
		if reflect.DeepEqual(original.Status, datacenter.Status) {
			return nil
		}

		// update the status
		return client.Status().Patch(ctx, datacenter, ctrlruntimeclient.MergeFrom(original))
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
		return client.Patch(ctx, kc, ctrlruntimeclient.MergeFrom(original))
	})
}
