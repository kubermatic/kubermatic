/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"fmt"

	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type StorageClassAnnotationFilter func(map[string]string) bool

func dataVolumeCreator(datavolume *cdiv1beta1.DataVolume) reconciling.NamedCDIv1beta1DataVolumeCreatorGetter {
	return func() (name string, create reconciling.CDIv1beta1DataVolumeCreator) {
		return datavolume.Name, func(dv *cdiv1beta1.DataVolume) (*cdiv1beta1.DataVolume, error) {
			dv.Spec = datavolume.Spec
			return dv, nil
		}
	}
}

func reconcilePreAllocatedDataVolumes(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) error {
	for _, d := range cluster.Spec.Cloud.Kubevirt.PreAllocatedDataVolumes {
		dv, err := createPreAllocatedDataVolume(d, cluster.Status.NamespaceName)
		if err != nil {
			return err
		}
		dvCreator := []reconciling.NamedCDIv1beta1DataVolumeCreatorGetter{
			dataVolumeCreator(dv),
		}
		if err := reconciling.ReconcileCDIv1beta1DataVolumes(ctx, dvCreator, cluster.Status.NamespaceName, client); err != nil {
			return fmt.Errorf("failed to reconcile Allocated DataVolume: %w", err)
		}
	}
	return nil
}

func createPreAllocatedDataVolume(dv kubermaticv1.PreAllocatedDataVolume, namespace string) (*cdiv1beta1.DataVolume, error) {
	dvSize, err := resource.ParseQuantity(dv.Size)
	if err != nil {
		return nil, err
	}
	return &cdiv1beta1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dv.Name,
			Namespace: namespace,
		},
		Spec: cdiv1beta1.DataVolumeSpec{
			Source: &cdiv1beta1.DataVolumeSource{
				HTTP: &cdiv1beta1.DataVolumeSourceHTTP{
					URL: dv.URL,
				},
			},
			PVC: &corev1.PersistentVolumeClaimSpec{
				StorageClassName: utilpointer.StringPtr(dv.StorageClass),
				AccessModes: []corev1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceStorage: dvSize},
				},
			},
		},
	}, nil
}

func ListStorageClasses(ctx context.Context, client ctrlruntimeclient.Client, annotationFilter StorageClassAnnotationFilter) (apiv2.StorageClassList, error) {
	storageClassList := storagev1.StorageClassList{}
	if err := client.List(ctx, &storageClassList); err != nil {
		return nil, err
	}

	res := apiv2.StorageClassList{}
	for _, sc := range storageClassList.Items {
		if annotationFilter == nil || annotationFilter(sc.Annotations) {
			res = append(res, apiv2.StorageClass{Name: sc.ObjectMeta.Name})
		}
	}
	return res, nil
}

func updateInfraStorageClassesInfo(ctx context.Context, spec *kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) error {
	kubeConfig, err := GetCredentialsForCluster(*spec, secretKeySelector)
	if err != nil {
		return err
	}
	client, _, err := NewClientWithRestConfig(kubeConfig)
	if err != nil {
		return err
	}
	storageClassList, err := ListStorageClasses(ctx, client, func(m map[string]string) bool {
		return m[InfraStorageClassAnnotation] == "true"
	})
	if err != nil {
		return err
	}
	existingStorageClassSet := sets.NewString(spec.Kubevirt.InfraStorageClasses...)

	for _, sc := range storageClassList {
		if !existingStorageClassSet.Has(sc.Name) {
			spec.Kubevirt.InfraStorageClasses = append(spec.Kubevirt.InfraStorageClasses, sc.Name)
		}
	}
	return nil
}
