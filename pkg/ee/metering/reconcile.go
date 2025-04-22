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
	"crypto/x509"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/ee/metering/prometheus"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/util/s3"
	"k8c.io/reconciler/pkg/reconciling"

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
	meteringName    = "metering"
	meteringVersion = "v1.2.2"
)

func getMeteringImage(overwriter registry.ImageRewriter) string {
	return registry.Must(overwriter(resources.RegistryQuay + "/kubermatic/metering:" + meteringVersion))
}

// ReconcileMeteringResources reconciles the metering related resources.
func ReconcileMeteringResources(ctx context.Context, client ctrlruntimeclient.Client, scheme *runtime.Scheme, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) error {
	overwriter := registry.GetImageRewriterFunc(cfg.Spec.UserCluster.OverwriteRegistry)

	if seed.Spec.Metering == nil || !seed.Spec.Metering.Enabled {
		return undeploy(ctx, client, seed.Namespace)
	}

	err := prometheus.ReconcilePrometheus(ctx, client, scheme, overwriter, seed)
	if err != nil {
		return fmt.Errorf("failed to reconcile metering prometheus: %w", err)
	}

	modifiers := []reconciling.ObjectModifier{
		common.VolumeRevisionLabelsModifierFactory(ctx, client),
		common.OwnershipModifierFactory(seed, scheme),
	}

	if err := reconcileMeteringReportConfigurations(ctx, client, seed, cfg.Spec.CABundle, overwriter, modifiers...); err != nil {
		return fmt.Errorf("failed to reconcile metering report configurations: %w", err)
	}

	return nil
}

func reconcileMeteringReportConfigurations(ctx context.Context, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed, caBundle corev1.TypedLocalObjectReference, overwriter registry.ImageRewriter, modifiers ...reconciling.ObjectModifier) error {
	if err := cleanupOrphanedReportingCronJobs(ctx, client, seed.Spec.Metering.ReportConfigurations, seed.Namespace); err != nil {
		return fmt.Errorf("failed to cleanup orphaned reporting cronjobs: %w", err)
	}

	config := lifecycle.NewConfiguration()
	var cronJobs []reconciling.NamedCronJobReconcilerFactory

	for reportName, reportConf := range seed.Spec.Metering.ReportConfigurations {
		cronJobs = append(cronJobs, CronJobReconciler(reportName, reportConf, caBundle.Name, overwriter, seed))

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
		seed.Namespace,
		client,
		modifiers...,
	); err != nil {
		return fmt.Errorf("failed to reconcile reporting cronjob: %w", err)
	}

	if len(config.Rules) == 0 {
		return nil
	}

	mc, bucket, err := getS3DataFromSeed(ctx, seed, client, caBundle.Name)
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
func cleanupOrphanedReportingCronJobs(ctx context.Context, client ctrlruntimeclient.Client, activeReports map[string]*kubermaticv1.MeteringReportConfiguration, namespace string) error {
	existingReportingCronJobs, err := fetchExistingReportingCronJobs(ctx, client, namespace)
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
func fetchExistingReportingCronJobs(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (*batchv1.CronJobList, error) {
	existingReportingCronJobs := &batchv1.CronJobList{}
	listOpts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(namespace),
		ctrlruntimeclient.ListOption(ctrlruntimeclient.MatchingLabels{common.ComponentLabel: meteringName}),
	}
	if err := client.List(ctx, existingReportingCronJobs, listOpts...); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to list reporting cronjobs: %w", err)
		}
	}
	return existingReportingCronJobs, nil
}

// undeploy removes all metering components expect the pvc used by prometheus.
func undeploy(ctx context.Context, client ctrlruntimeclient.Client, namespace string) error {
	var key types.NamespacedName

	existingReportingCronJobs, err := fetchExistingReportingCronJobs(ctx, client, namespace)
	if err != nil {
		return err
	}

	for _, cronJob := range existingReportingCronJobs.Items {
		key := types.NamespacedName{Namespace: cronJob.Namespace, Name: cronJob.Name}
		if err := cleanupResource(ctx, client, key, &batchv1.CronJob{}); err != nil {
			return fmt.Errorf("failed to cleanup metering CronJob: %w", err)
		}
	}

	key.Namespace = namespace
	key.Name = SecretName

	if err := cleanupResource(ctx, client, key, &corev1.Secret{}); err != nil {
		return fmt.Errorf("failed to cleanup metering s3 secret: %w", err)
	}

	// prometheus resources
	key.Name = prometheus.Name
	if err := cleanupResource(ctx, client, key, &corev1.Service{}); err != nil {
		return fmt.Errorf("failed to cleanup metering prometheus Service: %w", err)
	}
	if err := cleanupResource(ctx, client, key, &appsv1.StatefulSet{}); err != nil {
		return fmt.Errorf("failed to cleanup metering prometheus StatefulSet: %w", err)
	}
	if err := cleanupResource(ctx, client, key, &corev1.ConfigMap{}); err != nil {
		return fmt.Errorf("failed to cleanup metering prometheus ConfigMap: %w", err)
	}
	if err := cleanupResource(ctx, client, key, &rbacv1.ClusterRoleBinding{}); err != nil {
		return fmt.Errorf("failed to cleanup metering prometheus ClusterRoleBinding: %w", err)
	}
	if err := cleanupResource(ctx, client, key, &rbacv1.ClusterRole{}); err != nil {
		return fmt.Errorf("failed to cleanup metering prometheus ClusterRole: %w", err)
	}
	if err := cleanupResource(ctx, client, key, &corev1.ServiceAccount{}); err != nil {
		return fmt.Errorf("failed to cleanup metering prometheus ServiceAccount: %w", err)
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

func getS3DataFromSeed(ctx context.Context, seed *kubermaticv1.Seed, seedClient ctrlruntimeclient.Client, caBundleName string) (*minio.Client, string, error) {
	var s3secret corev1.Secret
	if err := seedClient.Get(ctx, types.NamespacedName{Name: SecretName, Namespace: seed.Namespace}, &s3secret); err != nil {
		return nil, "", err
	}

	s3endpoint := string(s3secret.Data[Endpoint])
	s3accessKeyID := string(s3secret.Data[AccessKey])
	s3secretAccessKey := string(s3secret.Data[SecretKey])

	// Fetch the ca-bundle
	var caBundle corev1.ConfigMap
	err := seedClient.Get(ctx, types.NamespacedName{Name: caBundleName, Namespace: seed.Namespace}, &caBundle)
	if err != nil {
		return nil, "", err
	}

	// Extract ca-bundle.pem from the ca-bundle configmap
	caBundleData, ok := caBundle.Data[resources.CABundleConfigMapKey]
	if !ok {
		return nil, "", fmt.Errorf("configMap does not contain key %q", resources.CABundleConfigMapKey)
	}

	// Create cert pool and append CA bundle
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM([]byte(caBundleData)); !ok {
		return nil, "", fmt.Errorf("failed to parse CA bundle")
	}

	mc, err := s3.NewClient(s3endpoint, s3accessKeyID, s3secretAccessKey, caCertPool)
	if err != nil {
		return nil, "", err
	}

	s3bucket := string(s3secret.Data[Bucket])

	return mc, s3bucket, nil
}
