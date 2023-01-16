package image

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	standardImageAnnotationKey = "kubevirt-images.k8c.io/standard-image"
	standardImageDefaultSize   = "11Gi"
)

// ReconcileStandardImages reconciles the DataVolumes for standard VM images if cloning is enabled.
func ReconcileStandardImages(ctx context.Context, dc *kubermaticv1.DatacenterSpecKubevirt, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	ir := newImageReconciler(ctx, logger, client,
		listStandardImages(dc),
		imageReconcilerStandard, standardImagesFilter, standardDataVolumeAnnotations())

	return ir.reconcile()
}

func standardDataVolumeAnnotations() map[string]string {
	return map[string]string{
		standardImageAnnotationKey:                   "true",
		dataVolumeDeleteAfterCompletionAnnotationKey: "false",
	}
}

func standardImagesFilter(annotations map[string]string) bool {
	return annotations != nil && annotations[standardImageAnnotationKey] != ""
}

// standardImages returns a list of PreAllocatedDataVolumes based on the list of standard images contained in the datacenter,
func listStandardImages(dc *kubermaticv1.DatacenterSpecKubevirt) []kubermaticv1.PreAllocatedDataVolume {
	dvs := make([]kubermaticv1.PreAllocatedDataVolume, 0)
	httpSource := dc.Images.HTTP

	imageSize := standardImageDefaultSize
	if httpSource.ImageCloning.DataVolumeSize != "" {
		imageSize = httpSource.ImageCloning.DataVolumeSize
	}

	// For this version, we handle only HTTP sources
	for os, osVersion := range httpSource.OperatingSystems {
		for version, url := range osVersion {
			dv := kubermaticv1.PreAllocatedDataVolume{
				Name:         fmt.Sprintf("%s-%s", os, version),
				URL:          url,
				Size:         imageSize,
				StorageClass: httpSource.ImageCloning.StorageClass,
			}
			dvs = append(dvs, dv)
		}
	}

	return dvs
}
