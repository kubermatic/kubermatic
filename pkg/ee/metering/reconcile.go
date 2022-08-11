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

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/ee/metering/prometheus"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (

	// legacy naming.
	meteringToolName = "kubermatic-metering"
	meteringDataName = "metering-data"

	meteringName      = "metering"
	meteringNamespace = resources.KubermaticNamespace
)

func getMeteringImage(overwriter registry.WithOverwriteFunc) string {
	return overwriter(resources.RegistryQuay) + "/kubermatic/metering:v1.0.0"
}

// ReconcileMeteringResources reconciles the metering related resources.
func ReconcileMeteringResources(ctx context.Context, client ctrlruntimeclient.Client, scheme *runtime.Scheme, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) error {
	overwriter := registry.GetOverwriteFunc(cfg.Spec.UserCluster.OverwriteRegistry)

	// ensure legacy components are removed
	err := cleanupLegacyMeteringResources(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to cleanup legacy metering components: %w", err)
	}

	if seed.Spec.Metering == nil || !seed.Spec.Metering.Enabled {
		return undeploy(ctx, client)
	}

	owner := common.OwnershipModifierFactory(seed, scheme)

	if err := reconciling.ReconcileNamespaces(ctx, []reconciling.NamedNamespaceCreatorGetter{
		meteringNamespaceCreator(),
	}, "", client); err != nil {
		return fmt.Errorf("failed to reconcile metering namespace: %w", err)
	}

	err = prometheus.ReconcilePrometheus(ctx, client, scheme, overwriter, seed)
	if err != nil {
		return fmt.Errorf("failed to reconcile metering prometheus: %w", err)
	}

	modifiers := []reconciling.ObjectModifier{
		common.VolumeRevisionLabelsModifierFactory(ctx, client),
		owner,
	}

	if err := reconcileMeteringReportConfigurations(ctx, client, seed, overwriter, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile metering report configurations: %w", err)
	}

	return nil
}

func reconcileMeteringReportConfigurations(ctx context.Context, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed, overwriter registry.WithOverwriteFunc, modifiers ...reconciling.ObjectModifier) error {
	if err := cleanupOrphanedReportingCronJobs(ctx, client, seed.Spec.Metering.ReportConfigurations); err != nil {
		return fmt.Errorf("failed to cleanup orphaned reporting cronjobs: %w", err)
	}

	config := lifecycle.NewConfiguration()
	var cronJobs []reconciling.NamedCronJobCreatorGetter

	for reportName, reportConf := range seed.Spec.Metering.ReportConfigurations {
		cronJobs = append(cronJobs, cronJobCreator(reportName, reportConf, overwriter))

		if reportConf.Retention != nil {
			config.Rules = append(config.Rules, lifecycle.Rule{
				ID:     reportName,
				Status: "Enabled",
				Expiration: lifecycle.Expiration{
					Days: lifecycle.ExpirationDays(*reportConf.Retention),
				},
				RuleFilter: lifecycle.Filter{
					Prefix: reportName,
				},
			})
		}
	}

	if err := reconciling.ReconcileCronJobs(
		ctx,
		cronJobs,
		resources.KubermaticNamespace,
		client,
		modifiers...,
	); err != nil {
		return fmt.Errorf("failed to reconcile reporting cronjob: %w", err)
	}

	mc, bucket, err := getS3DataFromSeed(ctx, client)
	if err != nil {
		return err
	}
	if err := mc.SetBucketLifecycle(ctx, bucket, config); err != nil {
		// Ignore conflicts in case lock after previous reconciliation process still exists.
		errResp := minio.ToErrorResponse(err)
		if errResp.StatusCode == 409 && errResp.Code == "OperationAborted" {
			return nil
		}
		return fmt.Errorf("failed to update bucket lifecycle: %w", err)
	}

	return nil
}

// cleanupOrphanedReportingCronJobs compares defined metering reports with existing reporting cronjobs and removes cronjobs with missing report configuration.
func cleanupOrphanedReportingCronJobs(ctx context.Context, client ctrlruntimeclient.Client, activeReports map[string]*kubermaticv1.MeteringReportConfiguration) error {
	existingReportingCronJobs, err := fetchExistingReportingCronJobs(ctx, client)
	if err != nil {
		return err
	}

	existingReportingCronJobNamedMap := make(map[string]batchv1.CronJob, len(existingReportingCronJobs.Items))
	for _, existingCronJob := range existingReportingCronJobs.Items {
		existingReportingCronJobNamedMap[existingCronJob.Name] = existingCronJob
	}

	activeReportsSet := sets.StringKeySet(activeReports)
	existingReportingCronJobsSet := sets.StringKeySet(existingReportingCronJobNamedMap)
	orphanedCronJobNames := existingReportingCronJobsSet.Difference(activeReportsSet)

	for cronJobName, existingCronJob := range existingReportingCronJobNamedMap {
		if orphanedCronJobNames.Has(cronJobName) {
			policy := metav1.DeletePropagationBackground
			delOpts := &ctrlruntimeclient.DeleteOptions{
				PropagationPolicy: &policy,
			}
			if err := client.Delete(ctx, &existingCronJob, delOpts); err != nil {
				return fmt.Errorf("failed to remove an orphaned reporting cronjob (%s): %w", cronJobName, err)
			}
		}
	}
	return nil
}

// fetchExistingReportingCronJobs returns a list of all existing reporting cronjobs.
func fetchExistingReportingCronJobs(ctx context.Context, client ctrlruntimeclient.Client) (*batchv1.CronJobList, error) {
	existingReportingCronJobs := &batchv1.CronJobList{}
	listOpts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(meteringNamespace),
		ctrlruntimeclient.ListOption(ctrlruntimeclient.HasLabels{meteringName}),
	}
	if err := client.List(ctx, existingReportingCronJobs, listOpts...); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to list reporting cronjobs: %w", err)
		}
	}
	return existingReportingCronJobs, nil
}

// undeploy removes all metering components expect the namespace and pvc.
func undeploy(ctx context.Context, client ctrlruntimeclient.Client) error {
	var key types.NamespacedName

	existingReportingCronJobs, err := fetchExistingReportingCronJobs(ctx, client)
	if err != nil {
		return err
	}

	for _, cronJob := range existingReportingCronJobs.Items {
		key := types.NamespacedName{Namespace: cronJob.Namespace, Name: cronJob.Name}
		if err := cleanupResource(ctx, client, key, &batchv1.CronJob{}); err != nil {
			return fmt.Errorf("failed to cleanup metering CronJob: %w", err)
		}
	}

	key.Namespace = meteringNamespace
	key.Name = SecretName

	if err := cleanupResource(ctx, client, key, &corev1.Secret{}); err != nil {
		return fmt.Errorf("failed to cleanup metering s3 secret: %w", err)
	}
	key.Name = resources.ImagePullSecretName
	if err := cleanupResource(ctx, client, key, &corev1.Secret{}); err != nil {
		return fmt.Errorf("failed to cleanup metering pull secret: %w", err)
	}

	// prometheus resources
	key.Name = prometheus.Name
	if err := cleanupResource(ctx, client, key, &corev1.Service{}); err != nil {
		return fmt.Errorf("failed to cleanup metering pull secret: %w", err)
	}

	if err := cleanupResource(ctx, client, key, &appsv1.StatefulSet{}); err != nil {
		return fmt.Errorf("failed to cleanup metering pull secret: %w", err)
	}

	if err := cleanupResource(ctx, client, key, &corev1.ConfigMap{}); err != nil {
		return fmt.Errorf("failed to cleanup metering pull secret: %w", err)
	}
	if err := cleanupResource(ctx, client, key, &rbacv1.ClusterRoleBinding{}); err != nil {
		return fmt.Errorf("failed to cleanup metering pull secret: %w", err)
	}
	if err := cleanupResource(ctx, client, key, &rbacv1.ClusterRole{}); err != nil {
		return fmt.Errorf("failed to cleanup metering pull secret: %w", err)
	}
	if err := cleanupResource(ctx, client, key, &corev1.ServiceAccount{}); err != nil {
		return fmt.Errorf("failed to cleanup metering pull secret: %w", err)
	}

	return nil
}

// cleanupLegacyMeteringResources removes all active parts of the legacy metering installation.
func cleanupLegacyMeteringResources(ctx context.Context, client ctrlruntimeclient.Client) error {
	key := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: meteringToolName}
	if err := cleanupResource(ctx, client, key, &appsv1.Deployment{}); err != nil {
		return fmt.Errorf("failed to cleanup metering Deployment: %w", err)
	}

	if err := cleanupResource(ctx, client, key, &rbacv1.ClusterRoleBinding{}); err != nil {
		return fmt.Errorf("failed to cleanup metering ClusterRoleBinding: %w", err)
	}

	if err := cleanupResource(ctx, client, key, &corev1.ServiceAccount{}); err != nil {
		return fmt.Errorf("failed to cleanup metering ServiceAccount: %w", err)
	}

	legacyReportingCronJobs := &batchv1.CronJobList{}
	listOpts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(resources.KubermaticNamespace),
		ctrlruntimeclient.ListOption(ctrlruntimeclient.HasLabels{meteringToolName}),
	}
	if err := client.List(ctx, legacyReportingCronJobs, listOpts...); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to list reporting cronjobs: %w", err)
		}
	}

	for _, cronJob := range legacyReportingCronJobs.Items {
		key = types.NamespacedName{Namespace: cronJob.Namespace, Name: cronJob.Name}
		if err := cleanupResource(ctx, client, key, &batchv1.CronJob{}); err != nil {
			return fmt.Errorf("failed to cleanup metering CronJob: %w", err)
		}
	}

	if err := cleanupResource(ctx, client, types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: meteringDataName}, &corev1.PersistentVolumeClaim{}); err != nil {
		return fmt.Errorf("failed to cleanup metering PersistentVolumeClaim: %w", err)
	}

	return nil
}

func cleanupResource(ctx context.Context, client ctrlruntimeclient.Client, key types.NamespacedName, obj ctrlruntimeclient.Object) error {
	if err := client.Get(ctx, key, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	return client.Delete(ctx, obj)
}
