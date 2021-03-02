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

package backup

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	utilpointer "k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// SharedVolumeName is the name of the `emptyDir` volume the initContainer
	// will write the backup to
	SharedVolumeName = "etcd-backup"
	// DefaultBackupContainerImage holds the default Image used for creating the etcd backups
	DefaultBackupContainerImage = "gcr.io/etcd-development/etcd"
	// DefaultBackupInterval defines the default interval used to create backups
	DefaultBackupInterval = "20m"
	// cronJobPrefix defines the prefix used for all backup cronjob names
	cronJobPrefix = "etcd-backup"
	// cleanupFinalizer defines the name for the finalizer to ensure we cleanup after we deleted a cluster
	cleanupFinalizer = "kubermatic.io/cleanup-backups"
	// backupCleanupJobLabel defines the label we use on all cleanup jobs
	backupCleanupJobLabel = "kubermatic-etcd-backup-cleaner"
	// clusterEnvVarKey defines the environment variable key for the cluster name
	clusterEnvVarKey = "CLUSTER"

	ControllerName = "kubermatic_backup_controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	log              *zap.SugaredLogger
	scheme           *runtime.Scheme
	workerName       string
	storeContainer   corev1.Container
	cleanupContainer corev1.Container
	// backupScheduleString is the cron string representing
	// the backupSchedule
	backupScheduleString string
	// backupContainerImage holds the image used for creating the etcd backup
	// It must be configurable to cover offline use cases
	backupContainerImage string
	// disabled means delete any existing back cronjob, rather than
	// ensuring they're installed. This is used to permanently delete the backup cronjobs
	// and disable the controller, usually in favor of the new one (../etcdbackup)
	disabled bool
	caBundle string

	recorder record.EventRecorder
	versions kubermatic.Versions
}

// Add creates a new Backup controller that is responsible for creating backupjobs
// for all managed user clusters
func Add(
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	storeContainer corev1.Container,
	cleanupContainer corev1.Container,
	backupSchedule time.Duration,
	backupContainerImage string,
	versions kubermatic.Versions,
	disabled bool,
	caBundleFile string,
) error {
	log = log.Named(ControllerName)
	if err := validateStoreContainer(storeContainer); err != nil {
		return err
	}
	backupScheduleString, err := parseDuration(backupSchedule)
	if err != nil {
		return fmt.Errorf("failed to parse backup duration: %v", err)
	}
	if backupContainerImage == "" {
		backupContainerImage = DefaultBackupContainerImage
	}

	caBundle, err := ioutil.ReadFile(caBundleFile)
	if err != nil {
		return fmt.Errorf("failed to read CA bundle file: %v", err)
	}

	reconciler := &Reconciler{
		Client:               mgr.GetClient(),
		log:                  log,
		workerName:           workerName,
		storeContainer:       storeContainer,
		cleanupContainer:     cleanupContainer,
		backupScheduleString: backupScheduleString,
		backupContainerImage: backupContainerImage,
		disabled:             disabled,
		recorder:             mgr.GetEventRecorderFor(ControllerName),
		versions:             versions,
		caBundle:             string(caBundle),
		scheme:               mgr.GetScheme(),
	}
	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	cronJobMapFn := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		if ownerRef := metav1.GetControllerOf(a); ownerRef != nil && ownerRef.Kind == kubermaticv1.ClusterKindName {
			return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: ownerRef.Name}}}
		}
		return nil
	})

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch Clusters: %v", err)
	}
	if err := c.Watch(&source.Kind{Type: &batchv1beta1.CronJob{}}, cronJobMapFn, predicate.ByNamespace(metav1.NamespaceSystem)); err != nil {
		return fmt.Errorf("failed to watch CronJobs: %v", err)
	}

	// Cleanup cleanup jobs...
	if err := mgr.Add(&runnableWrapper{
		f: func(ctx context.Context) {
			wait.UntilWithContext(ctx, reconciler.cleanupJobs, 30*time.Second)
		},
	}); err != nil {
		return fmt.Errorf("failed to add cleanup jobs runnable to mgr: %v", err)
	}

	return nil
}

type runnableWrapper struct {
	f func(context.Context)
}

func (w *runnableWrapper) Start(ctx context.Context) error {
	w.f(ctx)
	return nil
}

func (r *Reconciler) cleanupJobs(ctx context.Context) {
	log := r.log.Named("job_cleanup")

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", resources.AppLabelKey, backupCleanupJobLabel))
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	jobs := &batchv1.JobList{}
	if err := r.List(ctx, jobs, &ctrlruntimeclient.ListOptions{LabelSelector: selector}); err != nil {
		log.Errorw("failed to list jobs", "selector", selector.String(), zap.Error(err))
		utilruntime.HandleError(fmt.Errorf("failed to list jobs: %v", err))
		return
	}

	for _, job := range jobs.Items {
		if job.Status.Succeeded >= 1 && (job.Status.CompletionTime != nil && time.Since(job.Status.CompletionTime.Time).Minutes() > 5) {

			deletePropagationForeground := metav1.DeletePropagationForeground
			delOpts := &ctrlruntimeclient.DeleteOptions{
				PropagationPolicy: &deletePropagationForeground,
			}
			jobName := types.NamespacedName{Name: job.Name, Namespace: job.Namespace}
			if err := r.Delete(ctx, &job, delOpts); err != nil {
				log.Errorw(
					"Failed to delete cleanup job",
					zap.Error(err),
					"job_name", jobName,
				)
				utilruntime.HandleError(err)
				return
			}
			log.Infow("Deleted the cleanup job", "job_name", jobName)
		}
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: request.Name}, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	_, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionBackupControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return nil, r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	// Cluster got deleted - regardless if the cluster was ever running, we cleanup
	if cluster.DeletionTimestamp != nil {
		// Need to cleanup
		if sets.NewString(cluster.Finalizers...).Has(cleanupFinalizer) {
			if !r.disabled {
				if err := r.Create(ctx, r.cleanupJob(cluster)); err != nil {
					// Otherwise we end up in a loop when we are able to create the job but not
					// remove the finalizer.
					if !kerrors.IsAlreadyExists(err) {
						return err
					}
				}
			}

			oldCluster := cluster.DeepCopy()
			kuberneteshelper.RemoveFinalizer(cluster, cleanupFinalizer)
			if err := r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
				return fmt.Errorf("failed to update cluster after removing cleanup finalizer: %v", err)
			}
		}
		return nil
	}

	if cluster.Status.ExtendedHealth.Etcd != kubermaticv1.HealthStatusUp {
		log.Debug("Skipping because the cluster has no running etcd yet")
		return nil
	}

	// Always add the finalizer first
	if !kuberneteshelper.HasFinalizer(cluster, cleanupFinalizer) && !r.disabled {
		oldCluster := cluster.DeepCopy()
		kuberneteshelper.AddFinalizer(cluster, cleanupFinalizer)
		if err := r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return fmt.Errorf("failed to update cluster after adding cleanup finalizer: %v", err)
		}
	}

	if err := r.ensureCronJobSecrets(ctx, cluster); err != nil {
		return fmt.Errorf("failed to reconcile secrets: %v", err)
	}

	if err := r.ensureCronJobConfigMaps(ctx, cluster); err != nil {
		return fmt.Errorf("failed to reconcile configmaps: %v", err)
	}

	if r.disabled {
		return r.deleteCronJob(ctx, cluster)
	}
	return reconciling.ReconcileCronJobs(ctx, []reconciling.NamedCronJobCreatorGetter{r.cronjob(cluster)}, metav1.NamespaceSystem, r.Client)
}

func (r *Reconciler) getEtcdSecretName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s-etcd-client-certificate", cluster.Name)
}

func (r *Reconciler) ensureCronJobSecrets(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	secretName := r.getEtcdSecretName(cluster)

	getCA := func() (*triple.KeyPair, error) {
		return resources.GetClusterRootCA(ctx, cluster.Status.NamespaceName, r.Client)
	}

	creators := []reconciling.NamedSecretCreatorGetter{
		certificates.GetClientCertificateCreator(
			secretName,
			"backup",
			nil,
			resources.BackupEtcdClientCertificateCertSecretKey,
			resources.BackupEtcdClientCertificateKeySecretKey,
			getCA,
		),
	}

	return reconciling.ReconcileSecrets(ctx, creators, metav1.NamespaceSystem, r.Client, common.OwnershipModifierFactory(cluster, r.scheme))
}

func (r *Reconciler) ensureCronJobConfigMaps(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	name := resources.BackupCABundleConfigMapName(cluster)

	creators := []reconciling.NamedConfigMapCreatorGetter{
		caBundleConfigMapCreator(name, r.caBundle),
	}

	return reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespaceSystem, r.Client, common.OwnershipModifierFactory(cluster, r.scheme))
}

func (r *Reconciler) cleanupJob(cluster *kubermaticv1.Cluster) *batchv1.Job {
	cleanupContainer := r.cleanupContainer.DeepCopy()
	cleanupContainer.Env = append(cleanupContainer.Env, corev1.EnvVar{
		Name:  clusterEnvVarKey,
		Value: cluster.Name,
	})

	cleanupContainer.VolumeMounts = append(cleanupContainer.VolumeMounts, corev1.VolumeMount{
		Name:      "ca-bundle",
		MountPath: "/etc/ca-bundle",
		ReadOnly:  true,
	})

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("remove-cluster-backups-%s", cluster.Name),
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				resources.AppLabelKey: backupCleanupJobLabel,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          utilpointer.Int32Ptr(10),
			Completions:           utilpointer.Int32Ptr(1),
			Parallelism:           utilpointer.Int32Ptr(1),
			ActiveDeadlineSeconds: resources.Int64(30 * 60 * 60),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						*cleanupContainer,
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: SharedVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "ca-bundle",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: resources.BackupCABundleConfigMapName(cluster),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *Reconciler) cronjob(cluster *kubermaticv1.Cluster) reconciling.NamedCronJobCreatorGetter {
	return func() (string, reconciling.CronJobCreator) {
		return fmt.Sprintf("%s-%s", cronJobPrefix, cluster.Name), func(cronJob *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error) {
			gv := kubermaticv1.SchemeGroupVersion
			cronJob.OwnerReferences = []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, gv.WithKind(kubermaticv1.ClusterKindName)),
			}

			// Spec
			cronJob.Spec.Schedule = r.backupScheduleString
			cronJob.Spec.ConcurrencyPolicy = batchv1beta1.ForbidConcurrent
			cronJob.Spec.Suspend = utilpointer.BoolPtr(false)
			cronJob.Spec.SuccessfulJobsHistoryLimit = utilpointer.Int32Ptr(0)

			endpoints := etcd.GetClientEndpoints(cluster.Status.NamespaceName)
			image := r.backupContainerImage
			if !strings.Contains(image, ":") {
				image = image + ":" + etcd.ImageTag(cluster)
			}
			cronJob.Spec.JobTemplate.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:  "backup-creator",
					Image: image,
					Env: []corev1.EnvVar{
						{
							Name:  "ETCDCTL_API",
							Value: "3",
						},
						{
							Name:  "ETCDCTL_DIAL_TIMEOUT",
							Value: "3s",
						},
						{
							Name:  "ETCDCTL_CACERT",
							Value: "/etc/etcd/client/ca.crt",
						},
						{
							Name:  "ETCDCTL_CERT",
							Value: "/etc/etcd/client/backup-etcd-client.crt",
						},
						{
							Name:  "ETCDCTL_KEY",
							Value: "/etc/etcd/client/backup-etcd-client.key",
						},
					},
					Command: snapshotCommand(endpoints),
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      SharedVolumeName,
							MountPath: "/backup",
						},
						{
							Name:      "etcd-client-certificate",
							MountPath: "/etc/etcd/client",
						},
						{
							Name:      "ca-bundle",
							MountPath: "/etc/ca-bundle",
							ReadOnly:  true,
						},
					},
				},
			}

			storeContainer := r.storeContainer.DeepCopy()
			storeContainer.Env = append(storeContainer.Env, corev1.EnvVar{
				Name:  clusterEnvVarKey,
				Value: cluster.Name,
			})

			storeContainer.VolumeMounts = append(storeContainer.VolumeMounts, corev1.VolumeMount{
				Name:      "ca-bundle",
				MountPath: "/etc/ca-bundle/",
				ReadOnly:  true,
			})

			cronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
			cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{*storeContainer}
			cronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: SharedVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "etcd-client-certificate",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: r.getEtcdSecretName(cluster),
						},
					},
				},
				{
					Name: "ca-bundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.BackupCABundleConfigMapName(cluster),
							},
						},
					},
				},
			}

			return cronJob, nil
		}
	}
}

func (r *Reconciler) deleteCronJob(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	name := fmt.Sprintf("%s-%s", cronJobPrefix, cluster.Name)
	cj := &batchv1beta1.CronJob{}
	err := r.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: name}, cj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := r.Delete(ctx, cj, ctrlruntimeclient.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil && !kerrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to delete cronjob %s", name)
	}
	return nil
}

func parseDuration(interval time.Duration) (string, error) {
	scheduleString := fmt.Sprintf("@every %vm", interval.Round(time.Minute).Minutes())
	// We verify the validity of the scheduleString here, because the cronjob controller
	// only does that inside its sync loop, which means it is entirely possible to create
	// a cronJob with an invalid scheduleString
	// Refs:
	// https://github.com/kubernetes/kubernetes/blob/d02cf08e27f640f09ebd489e094176fd075f3463/pkg/controller/cronjob/cronjob_controller.go#L253
	// https://github.com/kubernetes/kubernetes/blob/d02cf08e27f640f09ebd489e094176fd075f3463/pkg/controller/cronjob/utils.go#L98
	_, err := cron.ParseStandard(scheduleString)
	if err != nil {
		return "", err
	}
	return scheduleString, nil
}

func validateStoreContainer(storeContainer corev1.Container) error {
	for _, volumeMount := range storeContainer.VolumeMounts {
		if volumeMount.Name == SharedVolumeName {
			return nil
		}
	}
	return fmt.Errorf("storeContainer does not have a mount for the shared volume %s", SharedVolumeName)
}

func snapshotCommand(etcdEndpoints []string) []string {
	cmd := []string{
		"/bin/sh",
		"-c",
	}
	script := &strings.Builder{}
	// Accordings to its godoc, this always returns a nil error
	_, _ = script.WriteString(
		`backupOrReportFailure() {
  echo "Creating backup"
  if ! eval $@; then
    echo "Backup creation failed"
    return 1
  fi
  echo "Successfully created backup, exiting"
  exit 0
}`)
	for _, endpoint := range etcdEndpoints {
		_, _ = script.WriteString(fmt.Sprintf("\nbackupOrReportFailure etcdctl --endpoints %s snapshot save /backup/snapshot.db", endpoint))
	}
	_, _ = script.WriteString("\necho \"Unable to create backup\"\nexit 1")
	cmd = append(cmd, script.String())
	return cmd
}
