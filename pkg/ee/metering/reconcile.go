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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	MeteringLabelKey = "kubermatic-metering"
)

func getMeteringImage(overwriter registry.WithOverwriteFunc) string {
	return overwriter(resources.RegistryQuay) + "/kubermatic/metering:6edff18"
}

func getMinioImage(overwriter registry.WithOverwriteFunc) string {
	return overwriter(resources.RegistryDocker) + "/minio/mc:RELEASE.2021-07-27T06-46-19Z"
}

// ReconcileMeteringResources reconciles the metering related resources.
func ReconcileMeteringResources(ctx context.Context, client ctrlruntimeclient.Client, scheme *runtime.Scheme, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) error {
	overwriter := registry.GetOverwriteFunc(cfg.Spec.UserCluster.OverwriteRegistry)

	if seed.Spec.Metering == nil || !seed.Spec.Metering.Enabled {
		return cleanupAllMeteringResources(ctx, client, seed.Spec.Metering)
	}

	if err := persistentVolumeClaimCreator(ctx, client, seed); err != nil {
		return fmt.Errorf("failed to reconcile metering PVC: %w", err)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, []reconciling.NamedServiceAccountCreatorGetter{
		serviceAccountCreator(),
	}, resources.KubermaticNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile metering ServiceAccounts: %w", err)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingCreatorGetter{
		clusterRoleBindingCreator(resources.KubermaticNamespace),
	}, "", client); err != nil {
		return fmt.Errorf("failed to reconcile metering ClusterRoleBindings: %w", err)
	}

	modifiers := []reconciling.ObjectModifier{
		common.VolumeRevisionLabelsModifierFactory(ctx, client),
		common.OwnershipModifierFactory(seed, scheme),
	}

	if err := reconcileMeteringReports(ctx, client, seed, overwriter, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile metering reports: %w", err)
	}

	if err := reconciling.ReconcileDeployments(ctx, []reconciling.NamedDeploymentCreatorGetter{
		deploymentCreator(seed, overwriter),
	}, resources.KubermaticNamespace, client, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile metering Deployment: %w", err)
	}

	return nil
}

func reconcileMeteringReports(ctx context.Context, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed, overwriter registry.WithOverwriteFunc, modifiers ...reconciling.ObjectModifier) error {
	if err := cleanupOrphanedReportingCronJobs(ctx, client, seed.Spec.Metering.ReportConfigurations); err != nil {
		return fmt.Errorf("failed to cleanup orphaned reporting cronjobs: %w", err)
	}
	for reportName, reportConf := range seed.Spec.Metering.ReportConfigurations {
		if err := reconciling.ReconcileCronJobs(
			ctx,
			[]reconciling.NamedCronJobCreatorGetter{cronJobCreator(seed.Name, reportName, reportConf, overwriter)},
			resources.KubermaticNamespace,
			client,
			modifiers...,
		); err != nil {
			return fmt.Errorf("failed to reconcile reporting cronjob: %w", err)
		}
	}
	return nil
}

// cleanupOrphanedReportingCronJobs compares defined metering reports with existing reporting cronjobs and removes cronjobs with missing report configuration.
func cleanupOrphanedReportingCronJobs(ctx context.Context, client ctrlruntimeclient.Client, activeReports map[string]*kubermaticv1.MeteringReportConfiguration) error {
	existingReportingCronJobs, err := fetchExistingReportingCronJobs(ctx, client)
	if err != nil {
		return err
	}

	existingReportingCronJobNamedMap := make(map[string]batchv1beta1.CronJob, len(existingReportingCronJobs.Items))
	for _, existingCronJob := range existingReportingCronJobs.Items {
		existingReportingCronJobNamedMap[existingCronJob.Name] = existingCronJob
	}

	activeReportsSet := sets.StringKeySet(activeReports)
	existingReportingCronJobsSet := sets.StringKeySet(existingReportingCronJobNamedMap)
	orphanedCronJobNames := existingReportingCronJobsSet.Difference(activeReportsSet)

	for cronJobName, existingCronJob := range existingReportingCronJobNamedMap {
		if orphanedCronJobNames.Has(cronJobName) {
			if err := client.Delete(ctx, &existingCronJob); err != nil {
				return fmt.Errorf("failed to remove an orphaned reporting cronjob (%s): %w", cronJobName, err)
			}
		}
	}
	return nil
}

// cleanupAllMeteringResources removes all active parts of the metering
// components, in case the admin disables the feature.
func cleanupAllMeteringResources(ctx context.Context, client ctrlruntimeclient.Client, meteringConfig *kubermaticv1.MeteringConfiguration) error {
	key := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: meteringToolName}
	if err := cleanupResource(ctx, client, key, &appsv1.Deployment{}); err != nil {
		return fmt.Errorf("failed to cleanup metering Deployment: %w", err)
	}

	existingReportingCronJobs, err := fetchExistingReportingCronJobs(ctx, client)
	if err != nil {
		return err
	}

	for _, cronJob := range existingReportingCronJobs.Items {
		key = types.NamespacedName{Namespace: cronJob.Namespace, Name: cronJob.Name}
		if err := cleanupResource(ctx, client, key, &batchv1beta1.CronJob{}); err != nil {
			return fmt.Errorf("failed to cleanup metering CronJob: %w", err)
		}
	}

	return nil
}

// fetchExistingReportingCronJobs returns a list of all existing reporting cronjobs.
func fetchExistingReportingCronJobs(ctx context.Context, client ctrlruntimeclient.Client) (*batchv1beta1.CronJobList, error) {
	existingReportingCronJobs := &batchv1beta1.CronJobList{}
	listOpts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(resources.KubermaticNamespace),
		ctrlruntimeclient.ListOption(ctrlruntimeclient.HasLabels{MeteringLabelKey}),
	}
	if err := client.List(ctx, existingReportingCronJobs, listOpts...); err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to list reporting cronjobs: %w", err)
		}
	}
	return existingReportingCronJobs, nil
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
