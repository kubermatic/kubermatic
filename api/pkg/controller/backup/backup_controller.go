package backup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	metricNamespace = "kubermatic"
	// SharedVolumeName is the name of the `emptyDir` volume the initContainer
	// will write the backup to
	SharedVolumeName = "etcd-backup"
	// DefaultBackupContainerImage holds the default Image used for creating the etcd backups
	DefaultBackupContainerImage = "gcr.io/etcd-development/etcd:" + etcd.ImageTag
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

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers prometheus.Gauge
}

// NewMetrics creates a new Metrics
// with default values initialized, so metrics always show up.
func NewMetrics() *Metrics {
	subsystem := "backup_controller"
	cm := &Metrics{
		Workers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running backup controller workers",
		}),
	}

	cm.Workers.Set(0)
	return cm
}

type Reconciler struct {
	workerName       string
	storeContainer   corev1.Container
	cleanupContainer corev1.Container
	// backupScheduleString is the cron string representing
	// the backupSchedule
	backupScheduleString string
	// backupContainerImage holds the image used for creating the etcd backup
	// It must be configurable to cover offline use cases
	backupContainerImage string

	metrics *Metrics
	ctrlruntimeclient.Client
	recorder record.EventRecorder
}

// Add creates a new Backup controller that is responsible for creating backupjobs
// for all managed user clusters
func Add(
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	storeContainer corev1.Container,
	cleanupContainer corev1.Container,
	backupSchedule time.Duration,
	backupContainerImage string,
	metrics *Metrics) error {
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

	if err := prometheus.Register(metrics.Workers); err != nil {
		return fmt.Errorf("failed to register worker metrics: %v", err)
	}

	reconciler := &Reconciler{
		workerName:           workerName,
		storeContainer:       storeContainer,
		cleanupContainer:     cleanupContainer,
		backupScheduleString: backupScheduleString,
		backupContainerImage: backupContainerImage,
		metrics:              metrics,
		Client:               mgr.GetClient(),
		recorder:             mgr.GetRecorder(ControllerName),
	}
	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	cronJobMapFn := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		// We only care about CronJobs that are in the kube-system namespace
		if a.Meta.GetNamespace() != metav1.NamespaceSystem {
			return nil
		}

		if ownerRef := metav1.GetControllerOf(a.Meta); ownerRef != nil && ownerRef.Kind == kubermaticv1.ClusterKindName {
			cluster := &kubermaticv1.Cluster{}
			err := mgr.GetClient().Get(context.Background(), types.NamespacedName{Name: ownerRef.Name}, cluster)
			if err != nil {
				if kerrors.IsNotFound(err) {
					return nil
				}
				glog.Errorf("failed to get cluster %q: %v", ownerRef.Name, err)
			}
			return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: ownerRef.Name}}}
		}
		return nil
	})}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch Clusters: %v", err)
	}
	if err := c.Watch(&source.Kind{Type: &batchv1beta1.CronJob{}}, cronJobMapFn); err != nil {
		return fmt.Errorf("failed to watch CronJobs: %v", err)
	}

	// Cleanup cleanup jobs...
	if err := mgr.Add(&runnableWrapper{
		f: func(stopCh <-chan struct{}) {
			wait.Until(reconciler.cleanupJobs, 30*time.Second, stopCh)
		},
	}); err != nil {
		return fmt.Errorf("failed to add cleanup jobs runnable to mgr: %v", err)
	}

	return nil
}

type runnableWrapper struct {
	f func(<-chan struct{})
}

func (w *runnableWrapper) Start(stopChan <-chan struct{}) error {
	w.f(stopChan)
	return nil
}

func (r *Reconciler) cleanupJobs() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", resources.AppLabelKey, backupCleanupJobLabel))
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	jobs := &batchv1.JobList{}
	if err := r.List(ctx, &ctrlruntimeclient.ListOptions{LabelSelector: selector}, jobs); err != nil {
		utilruntime.HandleError(fmt.Errorf("Failed to list jobs: %v", err))
		return
	}

	for _, job := range jobs.Items {
		if job.Status.Succeeded >= 1 && (job.Status.CompletionTime != nil && time.Since(job.Status.CompletionTime.Time).Minutes() > 5) {

			modifierForegroundDeletePropagation := func(deleteOpts *ctrlruntimeclient.DeleteOptions) {
				deletePropagationForeground := metav1.DeletePropagationForeground
				deleteOpts.PropagationPolicy = &deletePropagationForeground
			}
			if err := r.Delete(ctx, &job, modifierForegroundDeletePropagation); err != nil {
				utilruntime.HandleError(err)
				return
			}
		}
	}
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: request.Name}, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	err := r.reconcile(ctx, cluster)
	if err != nil {
		glog.Errorf("Failed to reconcile cluster %q: %v", request.Name, err)
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if cluster.Spec.Pause {
		glog.V(4).Infof("skipping paused cluster %s", cluster.Name)
		return nil
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return nil
	}

	glog.V(4).Infof("syncing cluster %s", cluster.Name)

	// Cluster got deleted - regardless if the cluster was ever running, we cleanup
	if cluster.DeletionTimestamp != nil {
		// Need to cleanup
		if sets.NewString(cluster.Finalizers...).Has(cleanupFinalizer) {
			if err := r.Create(ctx, r.cleanupJob(cluster)); err != nil {
				// Otherwise we end up in a loop when we are able to create the job but not
				// remove the finalizer.
				if !kerrors.IsAlreadyExists(err) {
					return err
				}
			}

			finalizers := sets.NewString(cluster.Finalizers...)
			finalizers.Delete(cleanupFinalizer)
			cluster.Finalizers = finalizers.List()
			if err := r.Update(ctx, cluster); err != nil {
				return fmt.Errorf("failed to update cluster after removing cleanup finalizer: %v", err)
			}
		}
		return nil
	}

	// Wait until we have a running cluster
	if cluster.Status.NamespaceName == "" || !cluster.Status.Health.Etcd {
		return nil
	}

	// Always add the finalizer first
	if !sets.NewString(cluster.Finalizers...).Has(cleanupFinalizer) {
		cluster.Finalizers = append(cluster.Finalizers, cleanupFinalizer)
		if err := r.Update(ctx, cluster); err != nil {
			return fmt.Errorf("failed to update cluster after adding cleanup finalizer: %v", err)
		}
	}

	if err := r.ensureCronJobSecret(ctx, cluster); err != nil {
		return fmt.Errorf("failed to create backup secret: %v", err)
	}

	return r.ensureCronJob(ctx, cluster)
}

func (r *Reconciler) getEtcdSecretName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s-etcd-client-certificate", cluster.Name)
}

func (r *Reconciler) ensureCronJobSecret(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	secretName := r.getEtcdSecretName(cluster)

	getCA := func() (*triple.KeyPair, error) {
		return resources.GetClusterRootCA(ctx, cluster, r.Client)
	}

	_, creator := certificates.GetClientCertificateCreator(
		secretName,
		"backup",
		nil,
		resources.BackupEtcdClientCertificateCertSecretKey,
		resources.BackupEtcdClientCertificateKeySecretKey,
		getCA,
	)()

	err := reconciling.EnsureNamedObject(
		ctx,
		types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: secretName},
		reconciling.SecretObjectWrapper(creator), r.Client, &corev1.Secret{})
	if err != nil {
		return fmt.Errorf("failed to ensure Secret %q: %v", secretName, err)
	}

	return nil
}

func (r *Reconciler) ensureCronJob(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	name, creator := r.cronjob(cluster)
	err := reconciling.EnsureNamedObject(ctx,
		types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: name},
		reconciling.CronJobObjectWrapper(creator),
		r.Client, &batchv1beta1.CronJob{})
	if err != nil {
		return fmt.Errorf("failed to ensure Cronjob kube-system/%q: %v", name, err)
	}

	return nil
}

func (r *Reconciler) cleanupJob(cluster *kubermaticv1.Cluster) *batchv1.Job {
	cleanupContainer := r.cleanupContainer.DeepCopy()
	cleanupContainer.Env = append(cleanupContainer.Env, corev1.EnvVar{
		Name:  clusterEnvVarKey,
		Value: cluster.Name,
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
			BackoffLimit:          int32Ptr(10),
			Completions:           int32Ptr(1),
			Parallelism:           int32Ptr(1),
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
					},
				},
			},
		},
	}
}

func (r *Reconciler) cronjob(cluster *kubermaticv1.Cluster) (string, reconciling.CronJobCreator) {
	return fmt.Sprintf("%s-%s", cronJobPrefix, cluster.Name), func(cronJob *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error) {
		gv := kubermaticv1.SchemeGroupVersion
		cronJob.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(cluster, gv.WithKind(kubermaticv1.ClusterKindName)),
		}

		// Spec
		cronJob.Spec.Schedule = r.backupScheduleString
		cronJob.Spec.ConcurrencyPolicy = batchv1beta1.ForbidConcurrent
		cronJob.Spec.Suspend = boolPtr(false)
		cronJob.Spec.SuccessfulJobsHistoryLimit = int32Ptr(int32(0))

		endpoints := etcd.GetClientEndpoints(cluster.Status.NamespaceName)
		cronJob.Spec.JobTemplate.Spec.Template.Spec.InitContainers = []corev1.Container{
			{
				Name:  "backup-creator",
				Image: r.backupContainerImage,
				Env: []corev1.EnvVar{
					{
						Name:  "ETCDCTL_API",
						Value: "3",
					},
				},
				Command: []string{
					"/usr/bin/time",
					"/usr/local/bin/etcdctl",
					"--endpoints", strings.Join(endpoints, ","),
					"--cacert", "/etc/etcd/client/ca.crt",
					"--cert", "/etc/etcd/client/backup-etcd-client.crt",
					"--key", "/etc/etcd/client/backup-etcd-client.key",
					"snapshot", "save", "/backup/snapshot.db",
				},
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

		storeContainer := r.storeContainer.DeepCopy()
		storeContainer.Env = append(storeContainer.Env, corev1.EnvVar{
			Name:  clusterEnvVarKey,
			Value: cluster.Name,
		})
		storeContainer.ImagePullPolicy = corev1.PullIfNotPresent
		storeContainer.TerminationMessagePath = corev1.TerminationMessagePathDefault
		storeContainer.TerminationMessagePolicy = corev1.TerminationMessageReadFile

		cronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{*storeContainer}
		cronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
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
						SecretName:  r.getEtcdSecretName(cluster),
						DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
					},
				},
			},
		}

		return cronJob, nil
	}

}

func boolPtr(b bool) *bool {
	return &b
}

func int32Ptr(i int32) *int32 {
	return &i
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
