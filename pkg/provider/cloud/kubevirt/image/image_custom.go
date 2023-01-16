package image

import (
	"context"

	"go.uber.org/zap"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const customImageTypeAnnotationKey = "kubevirt-images.k8c.io/os-type"

// ReconcileCustomImages reconciles the custom-disks from cluster.
func ReconcileCustomImages(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client, logger *zap.SugaredLogger) error {
	dvr := newImageReconciler(ctx, logger, client, listCustomImages(cluster), imageReconcilerCustom, customImagesFilter, customImagesAnnotations())
	return dvr.reconcile()
}

// customImages returns a list of PreAllocatedDataVolumes based on the list of standard images contained in the datacenter,
func listCustomImages(cluster *kubermaticv1.Cluster) []kubermaticv1.PreAllocatedDataVolume {
	return cluster.Spec.Cloud.Kubevirt.PreAllocatedDataVolumes
}

func customImagesFilter(annotations map[string]string) bool {
	return annotations != nil && annotations[customImageTypeAnnotationKey] != ""
}

func customImagesAnnotations() map[string]string {
	return map[string]string{
		dataVolumeDeleteAfterCompletionAnnotationKey: "false",
	}
}
