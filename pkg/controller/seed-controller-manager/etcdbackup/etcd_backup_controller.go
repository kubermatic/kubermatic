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

package etcdbackup

import (
	"context"
	"fmt"
	"time"

	cron "github.com/robfig/cron/v3"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	etcdbackup "k8c.io/kubermatic/v2/pkg/resources/etcd/backup"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// ControllerName name of etcd backup controller.
	ControllerName = "kkp-etcd-backup-controller"
	// DeleteAllBackupsFinalizer indicates that the backups still need to be deleted in the backend.
	DeleteAllBackupsFinalizer = "kubermatic.k8c.io/delete-all-backups"
	// DefaultBackupContainerImage holds the default Image used for creating the etcd backups.
	DefaultBackupContainerImage = "registry.k8s.io/etcd"

	// requeueAfter time after starting a job
	// should be the time after which a started job will usually have completed.
	assumedJobRuntime = 50 * time.Second

	// how long to keep succeeded and failed jobs around.
	// applies to both backup and backup delete jobs (except failed delete jobs, which will be restarted).
	// when the backup delete job is deleted, the corresponding etcdbackupconfig.status.currentBackups entry is also removed.
	succeededJobRetentionTime = 1 * time.Minute
	failedJobRetentionTime    = 10 * time.Minute

	// maximum number of simultaneously running backup delete jobs per BackupConfig.
	maxSimultaneousDeleteJobsPerConfig = 3
)

// Reconciler stores necessary components that are required to create etcd backups.
type Reconciler struct {
	ctrlruntimeclient.Client

	log        *zap.SugaredLogger
	scheme     *runtime.Scheme
	workerName string

	clock               clock.WithTickerAndDelayedExecution
	randStringGenerator func() string
	caBundle            resources.CABundle
	recorder            record.EventRecorder
	versions            kubermatic.Versions
	seedGetter          provider.SeedGetter
	configGetter        provider.KubermaticConfigurationGetter

	overwriteRegistry string
	etcdLauncherImage string
}

// Add creates a new Backup controller that is responsible for
// managing cluster etcd backups.
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,

	versions kubermatic.Versions,
	caBundle resources.CABundle,
	seedGetter provider.SeedGetter,
	configGetter provider.KubermaticConfigurationGetter,

	etcdLauncherImage string,
	overwriteRegistry string,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &Reconciler{
		Client:     client,
		log:        log,
		scheme:     mgr.GetScheme(),
		workerName: workerName,
		recorder:   mgr.GetEventRecorderFor(ControllerName),

		versions: versions,
		clock:    &clock.RealClock{},
		caBundle: caBundle,
		randStringGenerator: func() string {
			return rand.String(10)
		},
		seedGetter:   seedGetter,
		configGetter: configGetter,

		etcdLauncherImage: etcdLauncherImage,
		overwriteRegistry: overwriteRegistry,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.EtcdBackupConfig{}).
		Build(reconciler)

	return err
}

// Reconcile handle etcd backups reconciliation.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	seed, err := r.seedGetter()
	if err != nil {
		return reconcile.Result{}, err
	}

	config, err := r.configGetter(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	// this feature is not enabled for this seed, do nothing
	if !seed.IsEtcdAutomaticBackupEnabled() {
		return reconcile.Result{}, nil
	}

	log := r.log.With("request", request)
	log.Debug("Processing")

	backupConfig := &kubermaticv1.EtcdBackupConfig{}
	if err := r.Get(ctx, request.NamespacedName, backupConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: backupConfig.Spec.Cluster.Name}, cluster); err != nil {
		return reconcile.Result{}, err
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Cluster has no namespace name yet, skipping")
		return reconcile.Result{}, nil
	}

	if cluster.Status.Versions.ControlPlane == "" {
		log.Debug("Skipping because the cluster has no version status yet, skipping")
		return reconcile.Result{}, nil
	}

	log = r.log.With("cluster", cluster.Name, "backupConfig", backupConfig.Name)

	var suppressedError error

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionNone,
		func() (*reconcile.Result, error) {
			result, err := r.reconcile(ctx, log, backupConfig, cluster, seed, config)
			if apierrors.IsConflict(err) {
				// benign update conflict -- remember this so we can
				// suppress log.Error and event generation below
				suppressedError = err
			}
			return result, err
		},
	)
	if err != nil {
		if suppressedError != nil {
			// we know that err is a 1-element Aggregate containing just suppressedError
			// because we pass ClusterConditionNone above and thus ClusterReconcileWrapper()
			// couldn't have risen any additional errors
			log.Debugw("Benign update conflict error; will retry", zap.Error(suppressedError))
			result = &reconcile.Result{RequeueAfter: 30 * time.Second}
			err = nil
		} else {
			r.recorder.Event(backupConfig, corev1.EventTypeWarning, "ReconcilingError", err.Error())
			r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError",
				"failed to reconcile etcd backup config %q: %v", backupConfig.Name, err)
		}
	}

	if result == nil {
		result = &reconcile.Result{}
	}

	return *result, err
}

func (r *Reconciler) reconcile(
	ctx context.Context,
	log *zap.SugaredLogger,
	backupConfig *kubermaticv1.EtcdBackupConfig,
	cluster *kubermaticv1.Cluster,
	seed *kubermaticv1.Seed,
	config *kubermaticv1.KubermaticConfiguration,
) (*reconcile.Result, error) {
	data, err := r.getClusterTemplateData(ctx, cluster, seed, config, backupConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get template data: %w", err)
	}

	if err := r.ensureSecrets(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create backup secrets: %w", err)
	}

	if err := r.ensureConfigMaps(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create backup configmaps: %w", err)
	}

	var nextReconcile, totalReconcile *reconcile.Result

	if nextReconcile, err = r.ensurePendingBackupIsScheduled(ctx, backupConfig, cluster); err != nil {
		return nil, fmt.Errorf("failed to ensure next backup is scheduled: %w", err)
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.startPendingBackupJobs(ctx, data, backupConfig); err != nil {
		return nil, fmt.Errorf("failed to start pending and update running backups: %w", err)
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.startPendingBackupDeleteJobs(ctx, data, backupConfig); err != nil {
		return nil, fmt.Errorf("failed to start pending backup delete jobs: %w", err)
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.updateRunningBackupDeleteJobs(ctx, data, backupConfig); err != nil {
		return nil, fmt.Errorf("failed to update running backup delete jobs: %w", err)
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.deleteFinishedBackupJobs(ctx, log, backupConfig, cluster); err != nil {
		return nil, fmt.Errorf("failed to delete finished backup jobs: %w", err)
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.handleFinalization(ctx, backupConfig); err != nil {
		return nil, fmt.Errorf("failed to clean up EtcdBackupConfig: %w", err)
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	return totalReconcile, nil
}

func getBackupStoreContainer(cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) (*corev1.Container, error) {
	// a customized container is configured
	if cfg.Spec.SeedController.BackupStoreContainer != "" {
		return kuberneteshelper.ContainerFromString(cfg.Spec.SeedController.BackupStoreContainer)
	}

	return kuberneteshelper.ContainerFromString(defaulting.DefaultBackupStoreContainer)
}

func getBackupDeleteContainer(cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) (*corev1.Container, error) {
	// a customized container is configured
	if cfg.Spec.SeedController.BackupDeleteContainer != "" {
		return kuberneteshelper.ContainerFromString(cfg.Spec.SeedController.BackupDeleteContainer)
	}

	return kuberneteshelper.ContainerFromString(defaulting.DefaultBackupDeleteContainer)
}

func minReconcile(reconciles ...*reconcile.Result) *reconcile.Result {
	var result *reconcile.Result
	for _, r := range reconciles {
		if result == nil || (r != nil && r.RequeueAfter < result.RequeueAfter) {
			result = r
		}
	}
	return result
}

// ensure a backup is scheduled for the most recent backup time, according to the backup config's schedule.
// "schedule a backup" means set the scheduled time, backup name and job names of the corresponding element of backupConfig.Status.CurrentBackups.
func (r *Reconciler) ensurePendingBackupIsScheduled(ctx context.Context, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if backupConfig.DeletionTimestamp != nil || cluster.DeletionTimestamp != nil {
		// backupConfig is deleting. Don't schedule any new backups.
		return nil, nil
	}

	oldBackupConfig := backupConfig.DeepCopy()

	if len(backupConfig.Status.CurrentBackups) > 2*backupConfig.GetKeptBackupsCount() {
		// keeping track of many backups already, don't schedule new ones.
		if r.setBackupConfigCondition(
			backupConfig,
			kubermaticv1.EtcdBackupConfigConditionSchedulingActive,
			corev1.ConditionFalse,
			"TooManyBackups",
			"tracking too many backups; not scheduling new ones") {
			// condition changed, need to persist and generate an event
			if err := r.Status().Patch(ctx, backupConfig, ctrlruntimeclient.MergeFrom(oldBackupConfig)); err != nil {
				return nil, fmt.Errorf("failed to update backup config: %w", err)
			}
			r.recorder.Event(backupConfig, corev1.EventTypeWarning, "TooManyBackups", "tracking too many backups; not scheduling new ones")
		}

		return nil, nil
	} else if r.setBackupConfigCondition(
		backupConfig,
		kubermaticv1.EtcdBackupConfigConditionSchedulingActive,
		corev1.ConditionTrue,
		"",
		"") {
		// condition changed, need to persist and generate an event
		if err := r.Status().Patch(ctx, backupConfig, ctrlruntimeclient.MergeFrom(oldBackupConfig)); err != nil {
			return nil, fmt.Errorf("failed to update backup config: %w", err)
		}
		r.recorder.Event(backupConfig, corev1.EventTypeNormal, "NormalBackupCount", "backup count low enough; scheduling new backups")
	}

	var backupToSchedule *kubermaticv1.BackupStatus
	var requeueAfter time.Duration

	if backupConfig.Spec.Schedule == "" {
		// no schedule set => we need to schedule exactly one backup (if none was scheduled yet)
		if len(backupConfig.Status.CurrentBackups) > 0 {
			// backups scheduled; just take CurrentBackups[0] and wait for it
			durationToScheduledTime := backupConfig.Status.CurrentBackups[0].ScheduledTime.Sub(r.clock.Now())
			if durationToScheduledTime >= 0 {
				return &reconcile.Result{Requeue: true, RequeueAfter: durationToScheduledTime}, nil
			}
			return nil, nil
		}
		backupConfig.Status.CurrentBackups = []kubermaticv1.BackupStatus{{}}
		backupToSchedule = &backupConfig.Status.CurrentBackups[0]
		backupToSchedule.ScheduledTime = metav1.NewTime(r.clock.Now())
		backupToSchedule.BackupName = fmt.Sprintf("%s.db.gz", backupConfig.Name)
		requeueAfter = 0
	} else {
		// compute the pending (i.e. latest past) and the next (i.e. earliest future) backup time,
		// based on the most recent scheduled backup or, as a fallback, the backupConfig's creation time

		nextBackupTime := backupConfig.CreationTimestamp.Time

		if len(backupConfig.Status.CurrentBackups) > 0 {
			latestBackup := &backupConfig.Status.CurrentBackups[len(backupConfig.Status.CurrentBackups)-1]
			nextBackupTime = latestBackup.ScheduledTime.Time
		}

		schedule, err := parseCronSchedule(backupConfig.Spec.Schedule)
		if err != nil {
			return nil, fmt.Errorf("Failed to Parse Schedule %v: %w", backupConfig.Spec.Schedule, err)
		}

		now := r.clock.Now()

		var pendingBackupTime time.Time
		for nextBackupTime = schedule.Next(nextBackupTime); now.After(nextBackupTime); nextBackupTime = schedule.Next(nextBackupTime) {
			pendingBackupTime = nextBackupTime
		}

		if pendingBackupTime.IsZero() {
			// no pending backup time found, meaning all past backups have been scheduled already. Just wait for the next backup time
			return &reconcile.Result{Requeue: true, RequeueAfter: nextBackupTime.Sub(now)}, nil
		}

		backupConfig.Status.CurrentBackups = append(backupConfig.Status.CurrentBackups, kubermaticv1.BackupStatus{})
		backupToSchedule = &backupConfig.Status.CurrentBackups[len(backupConfig.Status.CurrentBackups)-1]
		backupToSchedule.ScheduledTime = metav1.NewTime(pendingBackupTime)
		backupToSchedule.BackupName = fmt.Sprintf("%s-%s.db.gz", backupConfig.Name, backupToSchedule.ScheduledTime.UTC().Format("2006-01-02t15-04-05"))
		requeueAfter = nextBackupTime.Sub(now)
	}

	backupToSchedule.JobName = r.limitNameLength(fmt.Sprintf("%s-backup-%s-create-%s", cluster.Name, backupConfig.Name, r.randStringGenerator()))
	backupToSchedule.DeleteJobName = r.limitNameLength(fmt.Sprintf("%s-backup-%s-delete-%s", cluster.Name, backupConfig.Name, r.randStringGenerator()))

	status := backupConfig.Status.DeepCopy()

	if err := r.Update(ctx, backupConfig); err != nil {
		return nil, fmt.Errorf("failed to update backup config: %w", err)
	}

	oldBackupConfig = backupConfig.DeepCopy()
	backupConfig.Status = *status
	if err := r.Status().Patch(ctx, backupConfig, ctrlruntimeclient.MergeFrom(oldBackupConfig)); err != nil {
		return nil, fmt.Errorf("failed to update backup status: %w", err)
	}

	return &reconcile.Result{Requeue: true, RequeueAfter: requeueAfter}, nil
}

func (r *Reconciler) limitNameLength(name string) string {
	if len(name) <= 63 {
		return name
	}
	randomness := r.randStringGenerator()
	return name[0:63-len(randomness)] + randomness
}

// setBackupConfigCondition sets a condition on a backupConfig, return true if the condition's
// status was changed. If the status has not changed, no other changes are made (i.e. the
// LastHeartbeatTime is not incremented if it would be the only change, to prevent us spamming
// the apiserver with tons of needless updates). This is the same behaviour that is used for
// ClusterConditions.
func (r *Reconciler) setBackupConfigCondition(backupConfig *kubermaticv1.EtcdBackupConfig, conditionType kubermaticv1.EtcdBackupConfigConditionType, status corev1.ConditionStatus, reason, message string) bool {
	newCondition := kubermaticv1.EtcdBackupConfigCondition{
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	oldCondition, hadCondition := backupConfig.Status.Conditions[conditionType]
	if hadCondition {
		conditionCopy := oldCondition.DeepCopy()

		// Reset the times before comparing
		conditionCopy.LastHeartbeatTime.Reset()
		conditionCopy.LastTransitionTime.Reset()

		if apiequality.Semantic.DeepEqual(*conditionCopy, newCondition) {
			return false
		}
	}

	now := metav1.Now()
	newCondition.LastHeartbeatTime = now
	newCondition.LastTransitionTime = oldCondition.LastTransitionTime
	if hadCondition && oldCondition.Status != status {
		newCondition.LastTransitionTime = now
	}

	if backupConfig.Status.Conditions == nil {
		backupConfig.Status.Conditions = map[kubermaticv1.EtcdBackupConfigConditionType]kubermaticv1.EtcdBackupConfigCondition{}
	}
	backupConfig.Status.Conditions[conditionType] = newCondition

	return true
}

// create any backup jobs that can be created, i.e. that don't exist yet while their scheduled time has arrived
// also update status of backups whose jobs have completed.
func (r *Reconciler) startPendingBackupJobs(ctx context.Context, data *resources.TemplateData, backupConfig *kubermaticv1.EtcdBackupConfig) (*reconcile.Result, error) {
	var returnReconcile *reconcile.Result

	for i := range backupConfig.Status.CurrentBackups {
		backup := &backupConfig.Status.CurrentBackups[i]
		if backup.BackupPhase != kubermaticv1.BackupStatusPhaseCompleted && backup.BackupPhase != kubermaticv1.BackupStatusPhaseFailed {
			if backup.BackupPhase == kubermaticv1.BackupStatusPhaseRunning {
				job := &batchv1.Job{}
				err := r.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: backup.JobName}, job)
				if err != nil {
					if !apierrors.IsNotFound(err) {
						return nil, fmt.Errorf("error getting job for backup %s: %w", backup.BackupName, err)
					}
					// job not found. Apparently deleted externally.
					backup.BackupPhase = kubermaticv1.BackupStatusPhaseFailed
					backup.BackupMessage = "backup job deleted externally"
					backup.BackupFinishedTime = metav1.NewTime(r.clock.Now())
				} else {
					if cond := getJobConditionIfTrue(job, batchv1.JobComplete); cond != nil {
						backup.BackupPhase = kubermaticv1.BackupStatusPhaseCompleted
						backup.BackupMessage = cond.Message
						backup.BackupFinishedTime = cond.LastTransitionTime
					} else if cond := getJobConditionIfTrue(job, batchv1.JobFailed); cond != nil {
						backup.BackupPhase = kubermaticv1.BackupStatusPhaseFailed
						backup.BackupMessage = cond.Message
						backup.BackupFinishedTime = cond.LastTransitionTime
					} else {
						// job still running
						returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: assumedJobRuntime})
					}
				}
			} else if backup.BackupPhase == "" && r.clock.Now().Sub(backup.ScheduledTime.Time) >= 0 && backupConfig.DeletionTimestamp == nil {
				job := etcdbackup.BackupJob(data, backupConfig, backup)
				if err := r.Create(ctx, job); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
					return nil, fmt.Errorf("error creating job for backup %s: %w", backup.BackupName, err)
				}
				backup.BackupPhase = kubermaticv1.BackupStatusPhaseRunning
				kuberneteshelper.AddFinalizer(backupConfig, DeleteAllBackupsFinalizer)
				returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: assumedJobRuntime})
			}
		}
	}

	status := backupConfig.Status.DeepCopy()

	if err := r.Update(ctx, backupConfig); err != nil {
		return nil, fmt.Errorf("failed to update backup config: %w", err)
	}

	oldBackupConfig := backupConfig.DeepCopy()
	backupConfig.Status = *status
	if err := r.Status().Patch(ctx, backupConfig, ctrlruntimeclient.MergeFrom(oldBackupConfig)); err != nil {
		return nil, fmt.Errorf("failed to update backup status: %w", err)
	}

	return returnReconcile, nil
}

// create any backup delete jobs that can be created, i.e. for all completed backups older than the last backupConfig.GetKeptBackupsCount() ones.
func (r *Reconciler) startPendingBackupDeleteJobs(ctx context.Context, data *resources.TemplateData, backupConfig *kubermaticv1.EtcdBackupConfig) (*reconcile.Result, error) {
	// one-shot backups are not deleted until their backupConfig is deleted
	if backupConfig.Spec.Schedule == "" && backupConfig.DeletionTimestamp == nil {
		return nil, nil
	}

	var backupsToDelete []*kubermaticv1.BackupStatus
	keepCount := backupConfig.GetKeptBackupsCount()
	if backupConfig.DeletionTimestamp != nil {
		keepCount = 0
	}
	kept := 0
	runningDeleteJobsCount := 0
	for i := len(backupConfig.Status.CurrentBackups) - 1; i >= 0; i-- {
		backup := &backupConfig.Status.CurrentBackups[i]
		if backup.DeletePhase == kubermaticv1.BackupStatusPhaseRunning {
			runningDeleteJobsCount++
		}
		if backup.BackupPhase == kubermaticv1.BackupStatusPhaseFailed && backup.DeletePhase == "" {
			backupsToDelete = append(backupsToDelete, backup)
		} else if backup.BackupPhase == kubermaticv1.BackupStatusPhaseCompleted {
			kept++
			if kept > keepCount && backup.DeletePhase == "" {
				backupsToDelete = append(backupsToDelete, backup)
			}
		}
	}

	oldBackupConfig := backupConfig.DeepCopy()

	modified := false
	for _, backup := range backupsToDelete {
		if runningDeleteJobsCount < maxSimultaneousDeleteJobsPerConfig {
			if err := r.createBackupDeleteJob(ctx, data, backupConfig, backup); err != nil {
				return nil, err
			}
			runningDeleteJobsCount++
			modified = true
		}
	}

	if modified {
		if err := r.Status().Patch(ctx, backupConfig, ctrlruntimeclient.MergeFrom(oldBackupConfig)); err != nil {
			return nil, fmt.Errorf("failed to update backup status: %w", err)
		}

		return &reconcile.Result{RequeueAfter: assumedJobRuntime}, nil
	}

	return nil, nil
}

func (r *Reconciler) createBackupDeleteJob(ctx context.Context, data *resources.TemplateData, backupConfig *kubermaticv1.EtcdBackupConfig, backup *kubermaticv1.BackupStatus) error {
	if data.EtcdBackupDeleteContainer() != nil {
		job := etcdbackup.BackupDeleteJob(data, backupConfig, backup)
		if err := r.Create(ctx, job); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
			return fmt.Errorf("error creating delete job for backup %s: %w", backup.BackupName, err)
		}
		backup.DeletePhase = kubermaticv1.BackupStatusPhaseRunning
	} else {
		// no deleteContainer configured. Just mark deletion as finished immediately.
		backup.DeletePhase = kubermaticv1.BackupStatusPhaseCompleted
		backup.DeleteFinishedTime = metav1.NewTime(r.clock.Now())
	}
	return nil
}

// update status of all delete jobs that have completed and are still marked as running.
func (r *Reconciler) updateRunningBackupDeleteJobs(ctx context.Context, data *resources.TemplateData, backupConfig *kubermaticv1.EtcdBackupConfig) (*reconcile.Result, error) {
	var returnReconcile *reconcile.Result

	oldBackupConfig := backupConfig.DeepCopy()

	// structs with the backup status and the DeleteMessage to set in case we restart the delete job
	type DeleteJobToRestart struct {
		backup        *kubermaticv1.BackupStatus
		deleteMessage string
	}
	var deleteJobsToRestart []DeleteJobToRestart
	runningDeleteJobsCount := 0
	for i := range backupConfig.Status.CurrentBackups {
		backup := &backupConfig.Status.CurrentBackups[i]
		if backup.DeletePhase == kubermaticv1.BackupStatusPhaseRunning {
			job := &batchv1.Job{}
			err := r.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: backup.DeleteJobName}, job)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					return nil, fmt.Errorf("error getting delete job for backup %s: %w", backup.BackupName, err)
				}
				// job not found. Apparently deleted, either externally or by us in a previous cycle.
				// recreate it. We want to see a finished delete job.
				deleteJobsToRestart = append(deleteJobsToRestart, DeleteJobToRestart{backup, "job was deleted, restarted it"})
			} else {
				if cond := getJobConditionIfTrue(job, batchv1.JobComplete); cond != nil {
					backup.DeletePhase = kubermaticv1.BackupStatusPhaseCompleted
					backup.DeleteMessage = cond.Message
					backup.DeleteFinishedTime = cond.LastTransitionTime
					returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: succeededJobRetentionTime})
				} else if cond := getJobConditionIfTrue(job, batchv1.JobFailed); cond != nil {
					// job failed. Delete and recreate it. Again, we want to see every delete job complete successfully because
					// delete jobs are the only things that know how to delete a backup.
					// Ideally jobs would support recreating failed or hanging pods themselves, but
					// they don't under all circumstances -- see https://github.com/kubernetes/kubernetes/issues/95431
					if err := r.Delete(ctx, job, ctrlruntimeclient.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil && !apierrors.IsNotFound(err) {
						return nil, fmt.Errorf("backup %s: failed to delete failed delete job %s: %w", backup.BackupName, backup.JobName, err)
					}
					deleteJobsToRestart = append(deleteJobsToRestart, DeleteJobToRestart{backup, fmt.Sprintf("Job failed: %s. Restarted.", cond.Message)})
				} else {
					// job still running
					runningDeleteJobsCount++
					returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: assumedJobRuntime})
				}
			}
		}
	}

	for _, deleteJobToRestart := range deleteJobsToRestart {
		if runningDeleteJobsCount < maxSimultaneousDeleteJobsPerConfig {
			if err := r.createBackupDeleteJob(ctx, data, backupConfig, deleteJobToRestart.backup); err != nil {
				return nil, err
			}
			deleteJobToRestart.backup.DeleteMessage = deleteJobToRestart.deleteMessage
			runningDeleteJobsCount++
			returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: assumedJobRuntime})
		}
	}

	if err := r.Status().Patch(ctx, backupConfig, ctrlruntimeclient.MergeFrom(oldBackupConfig)); err != nil {
		return nil, fmt.Errorf("failed to update backup status: %w", err)
	}

	return returnReconcile, nil
}

// Delete backup and delete jobs that have been finished for a while.
// For backups where both the backup and delete jobs have been deleted, delete the backup status entry too.
func (r *Reconciler) deleteFinishedBackupJobs(ctx context.Context, log *zap.SugaredLogger, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	var returnReconcile *reconcile.Result

	oldBackupConfig := backupConfig.DeepCopy()

	var newBackups []kubermaticv1.BackupStatus
	modified := false
	for _, backup := range backupConfig.Status.CurrentBackups {
		// if the backupConfig is being deleted and this backup hasn't even started yet,
		// we can just delete the backup from backupConfig.Status.CurrentBackups directly
		if backupConfig.DeletionTimestamp != nil && backup.BackupPhase == "" {
			// don't add backup to newBackups, which ends up deleting it from backupConfig.Status.CurrentBackups below
			modified = true
			continue
		}

		backupJobDeleted := false
		if !backup.BackupFinishedTime.IsZero() {
			var retentionTime time.Duration
			switch {
			case !backupConfig.DeletionTimestamp.IsZero():
				retentionTime = 0
			case backup.BackupPhase == kubermaticv1.BackupStatusPhaseCompleted:
				retentionTime = succeededJobRetentionTime
			default:
				retentionTime = failedJobRetentionTime
			}

			age := r.clock.Now().Sub(backup.BackupFinishedTime.Time)

			if age < retentionTime {
				// don't delete the job yet, but reconcile when the time has come to delete it
				returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: retentionTime - age})
			} else {
				// delete job
				job := &batchv1.Job{}

				err := r.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: backup.JobName}, job)
				switch {
				case apierrors.IsNotFound(err):
					backupJobDeleted = true
				case err == nil:
					err := r.Delete(ctx, job, ctrlruntimeclient.PropagationPolicy(metav1.DeletePropagationBackground))
					if err != nil && !apierrors.IsNotFound(err) {
						return nil, fmt.Errorf("backup %s: failed to delete backup job %s: %w", backup.BackupName, backup.JobName, err)
					}
					backupJobDeleted = true
				case !apierrors.IsNotFound(err):
					return nil, fmt.Errorf("backup %s: failed to get backup job %s: %w", backup.BackupName, backup.JobName, err)
				}
			}
		}

		deleteJobDeleted := false
		if !backup.DeleteFinishedTime.IsZero() {
			var retentionTime time.Duration
			if !backupConfig.DeletionTimestamp.IsZero() {
				retentionTime = 0
			} else {
				retentionTime = succeededJobRetentionTime
			}

			age := r.clock.Now().Sub(backup.DeleteFinishedTime.Time)

			if age < retentionTime {
				// don't delete the job yet, but reconcile when the time has come to delete it
				returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: retentionTime - age})
			} else {
				// delete job
				job := &batchv1.Job{}

				err := r.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: backup.DeleteJobName}, job)
				switch {
				case apierrors.IsNotFound(err):
					deleteJobDeleted = true
				case err == nil:
					err := r.Delete(ctx, job, ctrlruntimeclient.PropagationPolicy(metav1.DeletePropagationBackground))
					if err != nil && !apierrors.IsNotFound(err) {
						return nil, fmt.Errorf("backup %s: failed to delete job %s: %w", backup.BackupName, backup.DeleteJobName, err)
					}
					deleteJobDeleted = true
				case !apierrors.IsNotFound(err):
					return nil, fmt.Errorf("backup %s: failed to get job %s: %w", backup.BackupName, backup.DeleteJobName, err)
				}
			}
		}

		if backupJobDeleted && deleteJobDeleted {
			// don't add backup to newBackups, which ends up deleting it from backupConfig.Status.CurrentBackups below
			modified = true
			continue
		}

		newBackups = append(newBackups, backup)
	}

	if modified {
		backupConfig.Status.CurrentBackups = newBackups
		if err := r.Status().Patch(ctx, backupConfig, ctrlruntimeclient.MergeFrom(oldBackupConfig)); err != nil {
			return nil, fmt.Errorf("failed to update backup status: %w", err)
		}
	}

	return returnReconcile, nil
}

func (r *Reconciler) handleFinalization(ctx context.Context, backupConfig *kubermaticv1.EtcdBackupConfig) (*reconcile.Result, error) {
	if backupConfig.DeletionTimestamp == nil || len(backupConfig.Status.CurrentBackups) > 0 {
		return nil, nil
	}

	// in older releases, this code executed the legacy cleanup jobs, which were
	// part of the legacy backup controller; this new controller was meant to be
	// a drop-in replacement and so was able to also handle the cleanup stuff.
	// As it turned out, the delete container was never possible to be empty, so
	// the "compat mode" where we used the cleanup containers never happened;
	// That is why this function now only removes the finalizer if all backups
	// are gone.

	err := kuberneteshelper.TryRemoveFinalizer(ctx, r, backupConfig, DeleteAllBackupsFinalizer)

	return nil, err
}

func (r *Reconciler) ensureSecrets(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	secretName := etcdbackup.GetEtcdBackupSecretName(cluster)

	getCA := func() (*triple.KeyPair, error) {
		return resources.GetClusterRootCA(ctx, cluster.Status.NamespaceName, r)
	}

	creators := []reconciling.NamedSecretReconcilerFactory{
		certificates.GetClientCertificateReconciler(
			secretName,
			"backup",
			nil,
			resources.BackupEtcdClientCertificateCertSecretKey,
			resources.BackupEtcdClientCertificateKeySecretKey,
			getCA,
		),
	}

	return reconciling.ReconcileSecrets(ctx, creators, metav1.NamespaceSystem, r, modifier.Ownership(cluster, "", r.scheme))
}

func caBundleConfigMapName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s-ca-bundle", cluster.Name)
}

func (r *Reconciler) ensureConfigMaps(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	name := caBundleConfigMapName(cluster)

	creators := []reconciling.NamedConfigMapReconcilerFactory{
		certificates.CABundleConfigMapReconciler(name, r.caBundle),
	}

	return reconciling.ReconcileConfigMaps(ctx, creators, metav1.NamespaceSystem, r, modifier.Ownership(cluster, "", r.scheme))
}

func parseCronSchedule(scheduleString string) (cron.Schedule, error) {
	var validationErrors []error
	var schedule cron.Schedule

	// cron.Parse panics if schedule is empty
	if len(scheduleString) == 0 {
		return nil, fmt.Errorf("Schedule must be a non-empty valid Cron expression")
	}

	// adding a recover() around cron.Parse because it panics on empty string and is possible
	// that it panics under other scenarios as well.
	func() {
		defer func() {
			if r := recover(); r != nil {
				validationErrors = append(validationErrors, fmt.Errorf("(panic) invalid schedule: %v", r))
			}
		}()

		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if res, err := parser.Parse(scheduleString); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("invalid schedule: %w", err))
		} else {
			schedule = res
		}
	}()

	if len(validationErrors) > 0 {
		return nil, utilerrors.NewAggregate(validationErrors)
	}

	return schedule, nil
}

func getJobConditionIfTrue(job *batchv1.Job, condType batchv1.JobConditionType) *batchv1.JobCondition {
	if len(job.Status.Conditions) == 0 {
		return nil
	}
	for _, cond := range job.Status.Conditions {
		if cond.Type == condType && cond.Status == corev1.ConditionTrue {
			return cond.DeepCopy()
		}
	}
	return nil
}

func (r *Reconciler) getClusterTemplateData(ctx context.Context, cluster *kubermaticv1.Cluster, seed *kubermaticv1.Seed, config *kubermaticv1.KubermaticConfiguration, backupConfig *kubermaticv1.EtcdBackupConfig) (*resources.TemplateData, error) {
	destination := seed.GetEtcdBackupDestination(backupConfig.Spec.Destination)
	if destination == nil {
		return nil, fmt.Errorf("cannot find backup destination %q", backupConfig.Spec.Destination)
	}
	if destination.Credentials == nil {
		return nil, fmt.Errorf("credentials not set for backup destination %q", backupConfig.Spec.Destination)
	}

	storeContainer, err := getBackupStoreContainer(config, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd backup store container: %w", err)
	}

	deleteContainer, err := getBackupDeleteContainer(config, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd backup delete container: %w", err)
	}

	return resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithClient(r).
		WithCluster(cluster).
		WithSeed(seed.DeepCopy()).
		WithKubermaticConfiguration(config.DeepCopy()).
		WithVersions(r.versions).
		WithOverwriteRegistry(r.overwriteRegistry).
		WithEtcdLauncherImage(r.etcdLauncherImage).
		WithEtcdBackupStoreContainer(storeContainer).
		WithEtcdBackupDeleteContainer(deleteContainer).
		WithEtcdBackupDestination(destination).
		Build(), nil
}
