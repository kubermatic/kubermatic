package clusterdeletion

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupImageRegistryConfigs(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (deletedSomething bool, err error) {
	log = log.Named("ImageRegistryConfigCleanup")
	userClusterClient, err := d.userClusterClientGetter()
	if err != nil {
		return false, err
	}

	imageRegistryConfigs := &unstructured.UnstructuredList{}
	imageRegistryConfigs.SetAPIVersion("imageregistry.operator.openshift.io/v1")
	imageRegistryConfigs.SetKind("Config")

	if err := userClusterClient.List(ctx, &ctrlruntimeclient.ListOptions{}, imageRegistryConfigs); err != nil {
		if meta.IsNoMatchError(err) {
			log.Debug("Got a NoMatchError when listing ImageRegistryConfigs, skipping their cleanup")
			return false, nil
		}
		return false, fmt.Errorf("failed to list ImageRegistryConfigs: %v", err)
	}

	if len(imageRegistryConfigs.Items) == 0 {
		log.Debug("No ImageRegistryConfigs found, nothing to clean up")
		return false, nil
	}

	log.Debug("Found ImageRegistryConfigs", "num-image-registry-configs", len(imageRegistryConfigs.Items))

	for _, imageRegistry := range imageRegistryConfigs.Items {
		if err := userClusterClient.Delete(ctx, &imageRegistry); err != nil {
			return false, fmt.Errorf("failed to delete ImageRegistryConfig: %v", err)
		}
	}

	log.Debug("Successfully issued DELETE for all ImageRegistryConfigs")
	return true, nil
}
