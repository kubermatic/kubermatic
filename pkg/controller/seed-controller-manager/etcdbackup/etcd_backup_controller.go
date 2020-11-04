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
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	errors2 "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/rand"
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
	// ControllerName name of etcd backup controller.
	ControllerName = "kubermatic_etcd_backup_controller"

	// DeleteAllBackupsFinalizer indicates that the backups still need to be deleted in the backend
	DeleteAllBackupsFinalizer = "kubermatic.io/delete-all-backups"

	// BackupConfigNameLabelKey is the label key which should be used to name the BackupConfig a job belongs to
	BackupConfigNameLabelKey = "backupConfig"
	// DefaultBackupContainerImage holds the default Image used for creating the etcd backups
	DefaultBackupContainerImage = "gcr.io/etcd-development/etcd"
	// SharedVolumeName is the name of the `emptyDir` volume the initContainer
	// will write the backup to
	SharedVolumeName = "etcd-backup"
	// backupJobLabel defines the label we use on all backup jobs
	backupJobLabel = "kubermatic-etcd-backup"
	// clusterEnvVarKey defines the environment variable key for the cluster name
	clusterEnvVarKey = "CLUSTER"
	// backupToCreateEnvVarKey defines the environment variable key for the name of the backup to create
	backupToCreateEnvVarKey = "BACKUP_TO_CREATE"
	// backupToDeleteEnvVarKey defines the environment variable key for the name of the backup to delete
	backupToDeleteEnvVarKey = "BACKUP_TO_DELETE"
	// backupScheduleEnvVarKey defines the environment variable key for the backup schedule
	backupScheduleEnvVarKey = "BACKUP_SCHEDULE"
	// backupKeepCountEnvVarKey defines the environment variable key for the number of backups to keep
	backupKeepCountEnvVarKey = "BACKUP_KEEP_COUNT"
	// backupConfigEnvVarKey defines the environment variable key for the name of the backup configuration resource
	backupConfigEnvVarKey = "BACKUP_CONFIG"

	// requeueAfter time after starting a job
	// should be the time after which a started job will usually have completed
	assumedJobRuntime = 50 * time.Second

	// how long to keep succeeded and failed jobs around.
	// applies to both backup and backup delete jobs (except failed delete jobs, which will be restarted).
	// when the backup delete job is deleted, the corresponding etcdbackupconfig.status.currentBackups entry is also removed
	succeededJobRetentionTime = 1 * time.Minute
	failedJobRetentionTime    = 10 * time.Minute
)

// Reconciler stores necessary components that are required to create etcd backups
type Reconciler struct {
	log        *zap.SugaredLogger
	workerName string
	ctrlruntimeclient.Client
	storeContainer   *corev1.Container
	deleteContainer  *corev1.Container
	cleanupContainer *corev1.Container
	// backupContainerImage holds the image used for creating the etcd backup
	// It must be configurable to cover offline use cases
	backupContainerImage string
	clock                clock.Clock
	randStringGenerator  func() string
	recorder             record.EventRecorder
	versions             kubermatic.Versions
}

// Add creates a new Backup controller that is responsible for
// managing cluster etcd backups
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	storeContainer *corev1.Container,
	deleteContainer *corev1.Container,
	cleanupContainer *corev1.Container,
	backupContainerImage string,
	versions kubermatic.Versions,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	if backupContainerImage == "" {
		backupContainerImage = DefaultBackupContainerImage
	}

	reconciler := &Reconciler{
		log:                  log,
		Client:               client,
		workerName:           workerName,
		storeContainer:       storeContainer,
		deleteContainer:      deleteContainer,
		cleanupContainer:     cleanupContainer,
		backupContainerImage: backupContainerImage,
		recorder:             mgr.GetEventRecorderFor(ControllerName),
		versions:             versions,
		clock:                &clock.RealClock{},
		randStringGenerator: func() string {
			return rand.String(10)
		},
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.EtcdBackupConfig{}}, &handler.EnqueueRequestForObject{})
}

// Reconcile handle etcd backups reconciliation.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", request)
	log.Debug("Processing")

	backupConfig := &kubermaticv1.EtcdBackupConfig{}
	if err := r.Get(ctx, request.NamespacedName, backupConfig); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: backupConfig.Spec.Cluster.Name}, cluster); err != nil {
		return reconcile.Result{}, err
	}

	log = r.log.With("cluster", cluster.Name, "backupConfig", backupConfig.Name)

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionNone,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, backupConfig, cluster)
		},
	)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(backupConfig, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError",
			"failed to reconcile etcd backup config %q: %v", backupConfig.Name, err)
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if err := r.ensureEtcdClientSecret(ctx, cluster); err != nil {
		return nil, errors.Wrap(err, "failed to create backup secret")
	}

	var nextReconcile, totalReconcile *reconcile.Result
	var err error
	errorReconcile := &reconcile.Result{RequeueAfter: 1 * time.Minute}

	if nextReconcile, err = r.ensurePendingBackupIsScheduled(ctx, backupConfig, cluster); err != nil {
		return errorReconcile, errors.Wrap(err, "failed to ensure next backup is scheduled")
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.startPendingBackupJobs(ctx, backupConfig, cluster); err != nil {
		return errorReconcile, errors.Wrap(err, "failed to start pending and update running backups")
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.startPendingBackupDeleteJobs(ctx, backupConfig, cluster); err != nil {
		return errorReconcile, errors.Wrap(err, "failed to start pending backup delete jobs")
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.updateRunningBackupDeleteJobs(ctx, backupConfig, cluster); err != nil {
		return errorReconcile, errors.Wrap(err, "failed to update running backup delete jobs")
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.deleteFinishedBackupJobs(ctx, backupConfig, cluster); err != nil {
		return errorReconcile, errors.Wrap(err, "failed to delete finished backup jobs")
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	if nextReconcile, err = r.handleFinalization(ctx, backupConfig, cluster); err != nil {
		return errorReconcile, errors.Wrap(err, "failed to delete finished backup jobs")
	}

	totalReconcile = minReconcile(totalReconcile, nextReconcile)

	return totalReconcile, nil
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
// "schedule a backup" means set the scheduled time, backup name and job names of the corresponding element of backupConfig.Status.CurrentBackups
func (r *Reconciler) ensurePendingBackupIsScheduled(ctx context.Context, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if backupConfig.DeletionTimestamp != nil || cluster.DeletionTimestamp != nil {
		// backupConfig is deleting. Don't schedule any new backups.
		return nil, nil
	}

	if len(backupConfig.Status.CurrentBackups) > 2*backupConfig.GetKeptBackupsCount() {
		// keeping track of many backups already, don't schedule new ones.
		if r.setBackupConfigCondition(
			backupConfig,
			kubermaticv1.EtcdBackupConfigConditionSchedulingActive,
			corev1.ConditionFalse,
			"TooManyBackups",
			"tracking too many backups; not scheduling new ones") {

			// condition changed, need to persist and generate an event
			if err := r.Update(ctx, backupConfig); err != nil {
				return nil, errors.Wrap(err, "failed to update backup config")
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
		if err := r.Update(ctx, backupConfig); err != nil {
			return nil, errors.Wrap(err, "failed to update backup config")
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
		backupToSchedule.ScheduledTime = &metav1.Time{Time: r.clock.Now()}
		backupToSchedule.BackupName = backupConfig.Name
		requeueAfter = 0

	} else {
		// compute the pending (i.e. latest past) and the next (i.e. earliest future) backup time,
		// based on the most recent scheduled backup or, as a fallback, the backupConfig's creation time

		nextBackupTime := backupConfig.ObjectMeta.CreationTimestamp.Time

		if len(backupConfig.Status.CurrentBackups) > 0 {
			latestBackup := &backupConfig.Status.CurrentBackups[len(backupConfig.Status.CurrentBackups)-1]
			nextBackupTime = latestBackup.ScheduledTime.Time
		}

		schedule, err := parseCronSchedule(backupConfig.Spec.Schedule)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to Parse Schedule %v", backupConfig.Spec.Schedule)
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
		backupToSchedule.ScheduledTime = &metav1.Time{Time: pendingBackupTime}
		backupToSchedule.BackupName = fmt.Sprintf("%s-%s", backupConfig.Name, backupToSchedule.ScheduledTime.Format("2006-01-02t15-04-05"))
		requeueAfter = nextBackupTime.Sub(now)
	}

	backupToSchedule.JobName = r.limitNameLength(fmt.Sprintf("%s-backup-%s-create-%s", cluster.Name, backupConfig.Name, r.randStringGenerator()))
	backupToSchedule.DeleteJobName = r.limitNameLength(fmt.Sprintf("%s-backup-%s-delete-%s", cluster.Name, backupConfig.Name, r.randStringGenerator()))

	if err := r.Update(ctx, backupConfig); err != nil {
		return nil, errors.Wrap(err, "failed to update backup config")
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

// set a condition on a backupConfig, return true iff the condition's status was changed
func (r *Reconciler) setBackupConfigCondition(backupConfig *kubermaticv1.EtcdBackupConfig, conditionType kubermaticv1.EtcdBackupConfigConditionType, status corev1.ConditionStatus, reason, message string) bool {
	newCond := kubermaticv1.EtcdBackupConfigCondition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: metav1.Time{Time: r.clock.Now()},
		Reason:             reason,
		Message:            message,
	}
	for i := range backupConfig.Status.Conditions {
		cond := &backupConfig.Status.Conditions[i]
		if cond.Type == conditionType {
			if cond.Status == status {
				return false
			}
			*cond = newCond
			return true
		}
	}

	backupConfig.Status.Conditions = append(backupConfig.Status.Conditions, newCond)
	return true
}

// create any backup jobs that can be created, i.e. that don't exist yet while their scheduled time has arrived
// also update status of backups whose jobs have completed
func (r *Reconciler) startPendingBackupJobs(ctx context.Context, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	var returnReconcile *reconcile.Result

	for i := range backupConfig.Status.CurrentBackups {
		backup := &backupConfig.Status.CurrentBackups[i]
		if backup.BackupPhase != kubermaticv1.BackupStatusPhaseCompleted && backup.BackupPhase != kubermaticv1.BackupStatusPhaseFailed {
			if backup.BackupPhase == kubermaticv1.BackupStatusPhaseRunning {
				job := &batchv1.Job{}
				err := r.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: backup.JobName}, job)
				if err != nil {
					if !kerrors.IsNotFound(err) {
						return nil, errors.Wrapf(err, "error getting job for backup %s", backup.BackupName)
					}
					// job not found. Apparently deleted externally.
					backup.BackupPhase = kubermaticv1.BackupStatusPhaseFailed
					backup.BackupMessage = "backup job deleted externally"
					backup.BackupFinishedTime = &metav1.Time{Time: r.clock.Now()}
				} else {
					if cond := getJobConditionIfTrue(job, batchv1.JobComplete); cond != nil {
						backup.BackupPhase = kubermaticv1.BackupStatusPhaseCompleted
						backup.BackupMessage = cond.Message
						backup.BackupFinishedTime = &cond.LastTransitionTime
					} else if cond := getJobConditionIfTrue(job, batchv1.JobFailed); cond != nil {
						backup.BackupPhase = kubermaticv1.BackupStatusPhaseFailed
						backup.BackupMessage = cond.Message
						backup.BackupFinishedTime = &cond.LastTransitionTime
					} else {
						// job still running
						returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: assumedJobRuntime})
					}
				}

			} else if backup.BackupPhase == "" && r.clock.Now().Sub(backup.ScheduledTime.Time) >= 0 && backupConfig.DeletionTimestamp == nil {
				job := r.backupJob(backupConfig, cluster, backup)
				if err := r.Create(ctx, job); err != nil && !kerrors.IsAlreadyExists(err) {
					return nil, errors.Wrapf(err, "error creating job for backup %s", backup.BackupName)
				}
				backup.BackupPhase = kubermaticv1.BackupStatusPhaseRunning
				kuberneteshelper.AddFinalizer(backupConfig, DeleteAllBackupsFinalizer)
				returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: assumedJobRuntime})
			}
		}
	}

	if err := r.Update(ctx, backupConfig); err != nil {
		return nil, errors.Wrap(err, "failed to update backup config")
	}

	return returnReconcile, nil
}

// create any backup delete jobs that can be created, i.e. for all completed backups older than the last backupConfig.GetKeptBackupsCount() ones.
func (r *Reconciler) startPendingBackupDeleteJobs(ctx context.Context, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
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
	for i := len(backupConfig.Status.CurrentBackups) - 1; i >= 0; i-- {
		backup := &backupConfig.Status.CurrentBackups[i]
		if backup.BackupPhase == kubermaticv1.BackupStatusPhaseFailed && backup.DeletePhase == "" {
			backupsToDelete = append(backupsToDelete, backup)
		} else if backup.BackupPhase == kubermaticv1.BackupStatusPhaseCompleted {
			kept++
			if kept > keepCount && backup.DeletePhase == "" {
				backupsToDelete = append(backupsToDelete, backup)
			}
		}
	}

	modified := false
	for _, backup := range backupsToDelete {
		if err := r.createBackupDeleteJob(ctx, backupConfig, cluster, backup); err != nil {
			return nil, err
		}
		modified = true
	}

	if modified {
		if err := r.Update(ctx, backupConfig); err != nil {
			return nil, errors.Wrap(err, "failed to update backup config")
		}
		return &reconcile.Result{RequeueAfter: assumedJobRuntime}, nil
	}

	return nil, nil
}

func (r *Reconciler) createBackupDeleteJob(ctx context.Context, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster, backup *kubermaticv1.BackupStatus) error {
	if r.deleteContainer != nil {
		job := r.backupDeleteJob(backupConfig, cluster, backup)
		if err := r.Create(ctx, job); err != nil && !kerrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "error creating delete job for backup %s", backup.BackupName)
		}
		backup.DeletePhase = kubermaticv1.BackupStatusPhaseRunning
	} else {
		// no deleteContainer configured. Just mark deletion as finished immediately.
		backup.DeletePhase = kubermaticv1.BackupStatusPhaseCompleted
		backup.DeleteFinishedTime = &metav1.Time{Time: r.clock.Now()}
	}
	return nil
}

// update status of all delete jobs that have completed and are still marked as running
func (r *Reconciler) updateRunningBackupDeleteJobs(ctx context.Context, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	var returnReconcile *reconcile.Result

	for i := range backupConfig.Status.CurrentBackups {
		backup := &backupConfig.Status.CurrentBackups[i]
		if backup.DeletePhase == kubermaticv1.BackupStatusPhaseRunning {
			job := &batchv1.Job{}
			err := r.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: backup.DeleteJobName}, job)
			if err != nil {
				if !kerrors.IsNotFound(err) {
					return nil, errors.Wrapf(err, "error getting delete job for backup %s", backup.BackupName)
				}
				// job not found. Apparently deleted externally.
				// recreate it. We want to see a finished delete job.
				if err := r.createBackupDeleteJob(ctx, backupConfig, cluster, backup); err != nil {
					return nil, err
				}
				backup.DeleteMessage = "job deleted externally, restarted"
			} else {
				if cond := getJobConditionIfTrue(job, batchv1.JobComplete); cond != nil {
					backup.DeletePhase = kubermaticv1.BackupStatusPhaseCompleted
					backup.DeleteMessage = cond.Message
					backup.DeleteFinishedTime = &cond.LastTransitionTime
					returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: succeededJobRetentionTime})
				} else if cond := getJobConditionIfTrue(job, batchv1.JobFailed); cond != nil {
					// job failed. Delete and recreate it. Again, we want to see every delete job complete successfully because
					// delete jobs are the only things that know how to delete a backup.
					// Ideally jobs would support recreating failed or hanging pods themselves, but
					// they don't under all circumstances -- see https://github.com/kubernetes/kubernetes/issues/95431
					if err := r.Delete(ctx, job, ctrlruntimeclient.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil && !kerrors.IsNotFound(err) {
						return nil, errors.Wrapf(err, "backup %s: failed to delete failed delete job %s", backup.BackupName, backup.JobName)
					}
					if err := r.createBackupDeleteJob(ctx, backupConfig, cluster, backup); err != nil {
						return nil, err
					}
					backup.DeleteMessage = fmt.Sprintf("Job failed: %s. Restarted.", cond.Message)
					returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: assumedJobRuntime})
				} else {
					// job still running
					returnReconcile = minReconcile(returnReconcile, &reconcile.Result{RequeueAfter: assumedJobRuntime})
				}
			}
		}
	}

	if err := r.Update(ctx, backupConfig); err != nil {
		return nil, errors.Wrap(err, "failed to update backup config")
	}

	return returnReconcile, nil
}

// Delete backup and delete jobs that have been finished for a while.
// For backups where both the backup and delete jobs have been deleted, delete the backup status entry too.
func (r *Reconciler) deleteFinishedBackupJobs(ctx context.Context, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	var returnReconcile *reconcile.Result

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
		if backup.BackupFinishedTime != nil {
			var retentionTime time.Duration
			switch {
			case backupConfig.DeletionTimestamp != nil:
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
				case kerrors.IsNotFound(err):
					backupJobDeleted = true
				case err == nil:
					err := r.Delete(ctx, job, ctrlruntimeclient.PropagationPolicy(metav1.DeletePropagationBackground))
					if err != nil && !kerrors.IsNotFound(err) {
						return nil, errors.Wrapf(err, "backup %s: failed to delete backup job %s", backup.BackupName, backup.JobName)
					}
					backupJobDeleted = true
				case !kerrors.IsNotFound(err):
					return nil, errors.Wrapf(err, "backup %s: failed to get backup job %s", backup.BackupName, backup.JobName)
				}

			}
		}

		deleteJobDeleted := false
		if backup.DeleteFinishedTime != nil {
			var retentionTime time.Duration
			if backupConfig.DeletionTimestamp != nil {
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
				case kerrors.IsNotFound(err):
					deleteJobDeleted = true
				case err == nil:
					err := r.Delete(ctx, job, ctrlruntimeclient.PropagationPolicy(metav1.DeletePropagationBackground))
					if err != nil && !kerrors.IsNotFound(err) {
						return nil, errors.Wrapf(err, "backup %s: failed to delete job %s", backup.BackupName, backup.DeleteJobName)
					}
					deleteJobDeleted = true
				case !kerrors.IsNotFound(err):
					return nil, errors.Wrapf(err, "backup %s: failed to get job %s", backup.BackupName, backup.DeleteJobName)
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

		if err := r.Update(ctx, backupConfig); err != nil {
			return nil, errors.Wrap(err, "failed to update backup config")
		}
	}

	return returnReconcile, nil
}

func (r *Reconciler) handleFinalization(ctx context.Context, backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if backupConfig.DeletionTimestamp == nil || len(backupConfig.Status.CurrentBackups) > 0 {
		return nil, nil
	}

	canRemoveFinalizer := true

	if r.cleanupContainer != nil && r.deleteContainer == nil {
		// Need to run, track and delete a cleanup job

		cleanupJobName := fmt.Sprintf("%s-backup-%s-cleanup", cluster.Name, backupConfig.Name)

		if !backupConfig.Status.CleanupRunning {
			// job not started before. start it
			cleanupJob := r.cleanupJob(backupConfig, cluster, cleanupJobName)
			if err := r.Create(ctx, cleanupJob); err != nil && !kerrors.IsAlreadyExists(err) {
				return nil, errors.Wrapf(err, "error creating cleanup job (%v)", cleanupJobName)
			}
			backupConfig.Status.CleanupRunning = true
			canRemoveFinalizer = false

		} else {
			// job was started before. Re-acquire it and check completion status
			cleanupJob := &batchv1.Job{}
			err := r.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: cleanupJobName}, cleanupJob)
			if err == nil {
				jobSucceeded := nil != getJobConditionIfTrue(cleanupJob, batchv1.JobComplete)
				jobFailed := nil != getJobConditionIfTrue(cleanupJob, batchv1.JobFailed)
				if jobSucceeded || jobFailed {
					// job completed either way. delete it.
					if err := r.Delete(ctx, cleanupJob, ctrlruntimeclient.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil && !kerrors.IsNotFound(err) {
						return nil, errors.Wrapf(err, "failed to delete finished cleanup job %s", cleanupJobName)
					}
					if jobFailed {
						// job failed, restart it.
						canRemoveFinalizer = false
						cleanupJob := r.cleanupJob(backupConfig, cluster, cleanupJobName)
						if err := r.Create(ctx, cleanupJob); err != nil && !kerrors.IsAlreadyExists(err) {
							return nil, errors.Wrapf(err, "error recreating cleanup job (%v)", cleanupJobName)
						}
					}
				} else {
					// job still running
					canRemoveFinalizer = false
				}

			} else if !kerrors.IsNotFound(err) {
				return nil, errors.Wrapf(err, "error getting cleanup job previously started (%v)", cleanupJobName)
			}
			// err IsNotFound or job deleted successfully => fall through
		}
	}

	returnReconcile := &reconcile.Result{RequeueAfter: 30 * time.Second}

	if canRemoveFinalizer {
		kuberneteshelper.RemoveFinalizer(backupConfig, DeleteAllBackupsFinalizer)
		returnReconcile = nil
	}

	if err := r.Update(ctx, backupConfig); err != nil {
		return nil, errors.Wrap(err, "failed to update backup config")
	}

	return returnReconcile, nil
}

func (r *Reconciler) backupJob(backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster, backupStatus *kubermaticv1.BackupStatus) *batchv1.Job {
	storeContainer := r.storeContainer.DeepCopy()
	storeContainer.Env = append(
		storeContainer.Env,
		corev1.EnvVar{
			Name:  clusterEnvVarKey,
			Value: cluster.Name,
		},
		corev1.EnvVar{
			Name:  backupToCreateEnvVarKey,
			Value: backupStatus.BackupName,
		},
		corev1.EnvVar{
			Name:  backupScheduleEnvVarKey,
			Value: backupConfig.Spec.Schedule,
		},
		corev1.EnvVar{
			Name:  backupKeepCountEnvVarKey,
			Value: strconv.Itoa(backupConfig.GetKeptBackupsCount()),
		},
		corev1.EnvVar{
			Name:  backupConfigEnvVarKey,
			Value: backupConfig.Name,
		})

	job := r.jobBase(backupConfig, cluster, backupStatus.JobName)

	job.Spec.Template.Spec.Containers = []corev1.Container{*storeContainer}

	endpoints := etcd.GetClientEndpoints(cluster.Status.NamespaceName)
	image := r.backupContainerImage
	if !strings.Contains(image, ":") {
		image = image + ":" + etcd.ImageTag(cluster)
	}
	job.Spec.Template.Spec.InitContainers = []corev1.Container{
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
					Name:      r.getEtcdSecretName(cluster),
					MountPath: "/etc/etcd/client",
				},
			},
		},
	}

	job.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: SharedVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: r.getEtcdSecretName(cluster),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: r.getEtcdSecretName(cluster),
				},
			},
		},
	}

	return job
}

func (r *Reconciler) backupDeleteJob(backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster, backupStatus *kubermaticv1.BackupStatus) *batchv1.Job {
	deleteContainer := r.deleteContainer.DeepCopy()
	deleteContainer.Env = append(
		deleteContainer.Env,
		corev1.EnvVar{
			Name:  clusterEnvVarKey,
			Value: cluster.Name,
		},
		corev1.EnvVar{
			Name:  backupToDeleteEnvVarKey,
			Value: backupStatus.BackupName,
		},
		corev1.EnvVar{
			Name:  backupScheduleEnvVarKey,
			Value: backupConfig.Spec.Schedule,
		},
		corev1.EnvVar{
			Name:  backupKeepCountEnvVarKey,
			Value: strconv.Itoa(backupConfig.GetKeptBackupsCount()),
		},
		corev1.EnvVar{
			Name:  backupConfigEnvVarKey,
			Value: backupConfig.Name,
		})
	job := r.jobBase(backupConfig, cluster, backupStatus.DeleteJobName)
	job.Spec.Template.Spec.Containers = []corev1.Container{*deleteContainer}
	return job
}

func (r *Reconciler) cleanupJob(backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster, jobName string) *batchv1.Job {
	cleanupContainer := r.cleanupContainer.DeepCopy()
	cleanupContainer.Env = append(
		cleanupContainer.Env,
		corev1.EnvVar{
			Name:  clusterEnvVarKey,
			Value: cluster.Name,
		},
		corev1.EnvVar{
			Name:  backupConfigEnvVarKey,
			Value: backupConfig.Name,
		})
	job := r.jobBase(backupConfig, cluster, jobName)
	job.Spec.Template.Spec.Containers = []corev1.Container{*cleanupContainer}
	return job
}

func (r *Reconciler) jobBase(backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster, jobName string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				resources.AppLabelKey:    backupJobLabel,
				BackupConfigNameLabelKey: backupConfig.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				resources.GetClusterRef(cluster),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          utilpointer.Int32Ptr(3),
			Completions:           utilpointer.Int32Ptr(1),
			Parallelism:           utilpointer.Int32Ptr(1),
			ActiveDeadlineSeconds: resources.Int64(2 * 60),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
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

func (r *Reconciler) getEtcdSecretName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s-etcd-client-certificate", cluster.Name)
}

func (r *Reconciler) ensureEtcdClientSecret(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	secretName := r.getEtcdSecretName(cluster)

	getCA := func() (*triple.KeyPair, error) {
		return resources.GetClusterRootCA(ctx, cluster.Status.NamespaceName, r.Client)
	}

	_, creator := certificates.GetClientCertificateCreator(
		secretName,
		"backup",
		nil,
		resources.BackupEtcdClientCertificateCertSecretKey,
		resources.BackupEtcdClientCertificateKeySecretKey,
		getCA,
	)()

	wrappedCreator := reconciling.SecretObjectWrapper(creator)
	wrappedCreator = reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster))(wrappedCreator)

	err := reconciling.EnsureNamedObject(
		ctx,
		types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: secretName},
		wrappedCreator, r.Client, &corev1.Secret{}, false)
	if err != nil {
		return fmt.Errorf("failed to ensure Secret %q: %v", secretName, err)
	}

	return nil
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

		if res, err := cron.ParseStandard(scheduleString); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("invalid schedule: %v", err))
		} else {
			schedule = res
		}
	}()

	if len(validationErrors) > 0 {
		return nil, errors2.NewAggregate(validationErrors)
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
