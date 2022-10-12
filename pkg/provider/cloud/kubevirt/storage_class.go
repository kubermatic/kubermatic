package kubevirt

import (
	"context"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// InfraStorageClassAnnotation represents a storage class that should be initialized on user clusters
	infraStorageClassAnnotation = "kubevirt-initialization.k8c.io/initialize-sc"
)

type storageClassAnnotationFilter func(map[string]string) bool

// ListStorageClasses returns list of storage classes filtered by annotations
func ListStorageClasses(ctx context.Context, client ctrlruntimeclient.Client, annotationFilter storageClassAnnotationFilter) (apiv2.StorageClassList, error) {
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

func updateInfraStorageClassesInfo(ctx context.Context, spec *kubermaticv1.CloudSpec, client ctrlruntimeclient.Client) error {
	storageClassList, err := ListStorageClasses(ctx, client, func(m map[string]string) bool {
		return m[infraStorageClassAnnotation] == "true"
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
