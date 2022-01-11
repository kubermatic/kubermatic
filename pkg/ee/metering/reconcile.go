//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package metering

import (
	"context"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getMeteringImage(overwriter registry.WithOverwriteFunc) string {
	return overwriter(resources.RegistryQuay) + "/kubermatic/metering:v0.6"
}

func getMinioImage(overwriter registry.WithOverwriteFunc) string {
	return overwriter(resources.RegistryDocker) + "/minio/mc:RELEASE.2021-07-27T06-46-19Z"
}

// ReconcileMeteringResources reconciles the metering related resources.
func ReconcileMeteringResources(ctx context.Context, client ctrlruntimeclient.Client, cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed) error {
	overwriter := registry.GetOverwriteFunc(cfg.Spec.UserCluster.OverwriteRegistry)

	if seed.Spec.Metering == nil || !seed.Spec.Metering.Enabled {
		return cleanupMeteringResources(ctx, client)
	}

	if err := persistentVolumeClaimCreator(ctx, client, seed); err != nil {
		return fmt.Errorf("failed to reconcile metering PVC: %v", err)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, []reconciling.NamedServiceAccountCreatorGetter{
		serviceAccountCreator(),
	}, resources.KubermaticNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile metering ServiceAccounts: %v", err)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingCreatorGetter{
		clusterRoleBindingCreator(resources.KubermaticNamespace),
	}, "", client); err != nil {
		return fmt.Errorf("failed to reconcile metering ClusterRoleBindings: %v", err)
	}

	modifiers := []reconciling.ObjectModifier{
		common.VolumeRevisionLabelsModifierFactory(ctx, client),
	}
	if err := reconciling.ReconcileCronJobs(ctx, []reconciling.NamedCronJobCreatorGetter{
		cronJobCreator(seed.Name, overwriter),
	}, resources.KubermaticNamespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile metering CronJob: %v", err)
	}

	if err := reconciling.ReconcileDeployments(ctx, []reconciling.NamedDeploymentCreatorGetter{
		deploymentCreator(seed, overwriter),
	}, resources.KubermaticNamespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile metering Deployment: %v", err)
	}

	return nil
}

// cleanupMeteringResources removes active parts of the metering
// components, in case the admin disables the feature.
func cleanupMeteringResources(ctx context.Context, client ctrlruntimeclient.Client) error {
	key := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: meteringToolName}
	if err := cleanupResource(ctx, client, key, &appsv1.Deployment{}); err != nil {
		return fmt.Errorf("failed to cleanup metering Deployment: %v", err)
	}

	key.Name = meteringCronJobWeeklyName
	if err := cleanupResource(ctx, client, key, &batchv1beta1.CronJob{}); err != nil {
		return fmt.Errorf("failed to cleanup metering CronJob: %v", err)
	}

	return nil
}

func cleanupResource(ctx context.Context, client ctrlruntimeclient.Client, key types.NamespacedName, obj ctrlruntimeclient.Object) error {
	if err := client.Get(ctx, key, obj); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	return client.Delete(ctx, obj)
}
