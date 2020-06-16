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

package clusterdeletion

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const openshiftImageRegistryFinalizer = "imageregistry.operator.openshift.io/finalizer"

func (d *Deletion) cleanupImageRegistryConfigs(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (deletedSomething bool, err error) {
	log = log.Named("ImageRegistryConfigCleanup")
	userClusterClient, err := d.userClusterClientGetter()
	if err != nil {
		return false, err
	}

	if err := d.disableImageRegistryConfigsCreation(ctx, userClusterClient); err != nil {
		return false, err
	}

	imageRegistryConfigs := &unstructured.UnstructuredList{}
	imageRegistryConfigs.SetAPIVersion("imageregistry.operator.openshift.io/v1")
	imageRegistryConfigs.SetKind("Config")

	if err := userClusterClient.List(ctx, imageRegistryConfigs); err != nil {
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

	log.Debugw("Found ImageRegistryConfigs", "num-image-registry-configs", len(imageRegistryConfigs.Items))

	for _, imageRegistry := range imageRegistryConfigs.Items {
		if err := userClusterClient.Delete(ctx, &imageRegistry); err != nil {
			return false, fmt.Errorf("failed to delete ImageRegistryConfig %q: %v", imageRegistry.GetName(), err)
		}

		// The registry operator doesn't remove its finalizer when the registry has no storage config
		storageConfigEmpty, err := hasEmptyStorageConfig(imageRegistry)
		if err != nil {
			return false, fmt.Errorf("failed to check if storage config is empty for registryConfig %q: %v", imageRegistry.GetName(), err)
		}

		if !storageConfigEmpty {
			continue
		}

		if !kuberneteshelper.HasFinalizer(&imageRegistry, openshiftImageRegistryFinalizer) {
			continue
		}

		log.Debugw("ImageregistryConfig has no storage configured but finalizer, removing it", "name", imageRegistry.GetName())
		oldImageRegistry := imageRegistry.DeepCopy()
		kuberneteshelper.RemoveFinalizer(&imageRegistry, openshiftImageRegistryFinalizer)
		if err := userClusterClient.Patch(ctx, &imageRegistry, ctrlruntimeclient.MergeFrom(oldImageRegistry)); err != nil {
			return false, fmt.Errorf("failed to remove %q finalizer from imageRegistryConfig %q: %v", openshiftImageRegistryFinalizer, imageRegistry.GetName(), err)
		}

	}

	log.Debug("Successfully issued DELETE for all ImageRegistryConfigs")
	return true, nil
}

func (d *Deletion) disableImageRegistryConfigsCreation(ctx context.Context, userClusterClient ctrlruntimeclient.Client) error {
	// Prevent re-creation of ImageRegistryConfigs by using an intentionally defunct admissionWebhook
	creatorGetters := []reconciling.NamedValidatingWebhookConfigurationCreatorGetter{
		creationPreventingWebhook("imageregistry.operator.openshift.io", []string{"configs"}),
	}
	if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, creatorGetters, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to create ValidatingWebhookConfiguration to prevent creation of ImageRegistgryConfigs: %v", err)
	}

	return nil
}

// simplifiedRegistryConfig is a minimal subset of the ImageRegistryConfig that contains
// only the fields we care about. Since we only read from it but don't update its body,
// dropping the rest is fine.
type simplifiedRegistryConfig struct {
	Spec struct {
		Storage map[string]interface{} `json:"storage"`
	} `json:"spec"`
}

func hasEmptyStorageConfig(u unstructured.Unstructured) (bool, error) {
	rawData, err := u.MarshalJSON()
	if err != nil {
		return false, fmt.Errorf("failed to marshal Unstructured: %v", err)
	}

	registryConfig := &simplifiedRegistryConfig{}
	if err := json.Unmarshal(rawData, registryConfig); err != nil {
		return false, fmt.Errorf("failed to unmarshal registry config: %v", err)
	}
	return len(registryConfig.Spec.Storage) == 0, nil
}
