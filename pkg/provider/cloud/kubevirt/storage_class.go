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

package kubevirt

import (
	"context"

	apiv2 "k8c.io/kubermatic/sdk/v2/api/v2"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type storageClassAnnotationFilter func(map[string]string) bool

// ListStorageClasses returns list of storage classes filtered by annotations.
func ListStorageClasses(ctx context.Context, client ctrlruntimeclient.Client, annotationFilter storageClassAnnotationFilter) (apiv2.StorageClassList, error) {
	storageClassList := storagev1.StorageClassList{}
	if err := client.List(ctx, &storageClassList); err != nil {
		return nil, err
	}

	res := apiv2.StorageClassList{}
	for _, sc := range storageClassList.Items {
		if annotationFilter == nil || annotationFilter(sc.Annotations) {
			res = append(res, apiv2.StorageClass{Name: sc.Name})
		}
	}
	return res, nil
}

func updateInfraStorageClassesInfo(ctx context.Context, client ctrlruntimeclient.Client, spec *kubermaticv1.KubevirtCloudSpec, dc *kubermaticv1.DatacenterSpecKubevirt) error {
	// TODO(mq): this function seems a bit out of place as there could be use cases where storage class exists in the cluster
	// and doesn't exist in the infra cluster such local path provisioner. Thus should be removed in the future.

	// If the Namespaced mode is enabled, kkp has no access on cluster-scoped resources such as storage classes. Thus,
	// this function will fail to list storage classes which means it will fail to create the cluster.
	if dc.NamespacedMode != nil && dc.NamespacedMode.Enabled {
		// considering the storage classes in the dc object as the only canonical truth and skip filtering storage classes.
		spec.StorageClasses = dc.InfraStorageClasses
		return nil
	}

	infraStorageClassList, err := ListStorageClasses(ctx, client, nil)
	if err != nil {
		return err
	}
	existingInfraStorageClassSet := make(sets.Set[string], len(infraStorageClassList))
	for _, isc := range infraStorageClassList {
		existingInfraStorageClassSet.Insert(isc.Name)
	}

	// Cluster will contain a list with only the StorageClasses
	// that are in the DC configuration and also exist in the infra KubeVirt cluster.
	storageClasses := make([]kubermaticv1.KubeVirtInfraStorageClass, 0)
	for _, sc := range dc.InfraStorageClasses {
		// if StorageClass exists in infra, keep it
		if existingInfraStorageClassSet.Has(sc.Name) {
			storageClasses = append(storageClasses, sc)
		}
	}
	spec.StorageClasses = storageClasses
	return nil
}
