package backup

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"

	"k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	informersbatchv1 "k8s.io/client-go/informers/batch/v1"
	informersbatchv1beta1 "k8s.io/client-go/informers/batch/v1beta1"
	corev1informer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listersbatchv1 "k8s.io/client-go/listers/batch/v1"
	listersbatchv1beta1 "k8s.io/client-go/listers/batch/v1beta1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/cert/triple"
	"k8s.io/client-go/util/workqueue"
)

const (
	metricNamespace = "kubermatic"
	// SharedVolumeName is the name of the `emptyDir` volume the initContainer
	// will write the backup to
	SharedVolumeName = "etcd-backup"
	// DefaultBackupContainerImage holds the default Image used for creating the etcd backups
	DefaultBackupContainerImage = "quay.io/coreos/etcd:v3.3"
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
)

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers                  prometheus.Gauge
	CronJobCreationTimestamp *prometheus.GaugeVec
	CronJobUpdateTimestamp   *prometheus.GaugeVec
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
		CronJobCreationTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "cronjob_creation_timestamp_seconds",
			Help:      "The timestamp at which a cronjob for a given cluster was created",
		}, []string{"cluster"}),
		CronJobUpdateTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "cronjob_update_timestamp_seconds",
			Help:      "The timestamp at which a cronjob for a given cluster was last updated",
		}, []string{"cluster"}),
	}

	cm.Workers.Set(0)
	return cm
}

// Controller stores all components required to create backups
type Controller struct {
	storeContainer   corev1.Container
	cleanupContainer corev1.Container
	// backupScheduleString is the cron string representing
	// the backupSchedule
	backupScheduleString string
	// backupContainerImage holds the image used for creating the etcd backup
	// It must be configurable to cover offline use cases
	backupContainerImage string
	// workerName holds the name of this worker, only clusters with matching `.Spec.WorkerName` will be worked on
	workerName string
	metrics    *Metrics

	queue            workqueue.RateLimitingInterface
	kubermaticClient kubermaticclientset.Interface
	kubernetesClient kubernetes.Interface
	clusterLister    kubermaticv1lister.ClusterLister
	cronJobLister    listersbatchv1beta1.CronJobLister
	jobLister        listersbatchv1.JobLister
	secretLister     corev1lister.SecretLister
}

// New creates a new Backup controller that is responsible for creating backupjobs
// for all managed user clusters
func New(
	storeContainer corev1.Container,
	cleanupContainer corev1.Container,
	backupSchedule time.Duration,
	backupContainerImage string,
	workerName string,
	metrics *Metrics,
	kubermaticClient kubermaticclientset.Interface,
	kubernetesClient kubernetes.Interface,
	clusterInformer kubermaticv1informers.ClusterInformer,
	cronJobInformer informersbatchv1beta1.CronJobInformer,
	jobInformer informersbatchv1.JobInformer,
	secretInformer corev1informer.SecretInformer,
) (*Controller, error) {
	if err := validateStoreContainer(storeContainer); err != nil {
		return nil, err
	}
	backupScheduleString, err := parseDuration(backupSchedule)
	if err != nil {
		return nil, fmt.Errorf("failed to parse backup duration: %v", err)
	}
	if backupContainerImage == "" {
		backupContainerImage = DefaultBackupContainerImage
	}
	c := &Controller{
		queue:                workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Backup"),
		kubermaticClient:     kubermaticClient,
		kubernetesClient:     kubernetesClient,
		backupScheduleString: backupScheduleString,
		storeContainer:       storeContainer,
		cleanupContainer:     cleanupContainer,
		backupContainerImage: backupContainerImage,
		workerName:           workerName,
		metrics:              metrics,
	}

	prometheus.MustRegister(metrics.Workers)
	prometheus.MustRegister(metrics.CronJobCreationTimestamp)
	prometheus.MustRegister(metrics.CronJobUpdateTimestamp)

	cronJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.handleObject(obj.(*batchv1beta1.CronJob))
		},
		UpdateFunc: func(_, new interface{}) {
			c.handleObject(new.(*batchv1beta1.CronJob))
		},
		DeleteFunc: func(obj interface{}) {
			cronJob, ok := obj.(*batchv1beta1.CronJob)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					runtime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				cronJob, ok = tombstone.Obj.(*batchv1beta1.CronJob)
				if !ok {
					runtime.HandleError(fmt.Errorf("tombstone contained object that is not a cronJob %#v", obj))
					return
				}
			}
			c.handleObject(cronJob)
		},
	})

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueue(obj.(*kubermaticv1.Cluster))
		},
		UpdateFunc: func(_, new interface{}) {
			c.enqueue(new.(*kubermaticv1.Cluster))
		},
		DeleteFunc: func(obj interface{}) {
			cluster, ok := obj.(*kubermaticv1.Cluster)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					runtime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				cluster, ok = tombstone.Obj.(*kubermaticv1.Cluster)
				if !ok {
					runtime.HandleError(fmt.Errorf("tombstone contained object that is not a Cluster %#v", obj))
					return
				}
			}
			c.enqueue(cluster)
		},
	})
	c.clusterLister = clusterInformer.Lister()
	c.cronJobLister = cronJobInformer.Lister()
	c.jobLister = jobInformer.Lister()
	c.secretLister = secretInformer.Lister()
	return c, nil
}

func (c *Controller) handleObject(obj interface{}) {
	object, ok := obj.(metav1.Object)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
		return
	}

	// We only care about CronJobs that are in the kube-system namespace
	if object.GetNamespace() != metav1.NamespaceSystem {
		return
	}

	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil && ownerRef.Kind == kubermaticv1.ClusterKindName {
		cluster, err := c.clusterLister.Get(ownerRef.Name)
		if err != nil {
			glog.V(4).Infof("Ignoring orphaned object '%s' from cluster '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}
		c.enqueue(cluster)
		return
	}
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	// Cleanup cleanup jobs...
	go wait.Until(c.cleanupJobs, 30*time.Second, stopCh)

	c.metrics.Workers.Set(float64(workerCount))
	<-stopCh
}

func (c *Controller) cleanupJobs() {
	selector, err := labels.Parse(fmt.Sprintf("%s=%s", resources.AppLabelKey, backupCleanupJobLabel))
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	jobList, err := c.jobLister.List(selector)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Failed to list jobs: %v", err))
		return
	}

	for _, job := range jobList {
		if job.Status.Succeeded >= 1 && (job.Status.CompletionTime != nil && time.Since(job.Status.CompletionTime.Time).Minutes() > 5) {
			propagation := metav1.DeletePropagationForeground
			if err := c.kubernetesClient.BatchV1().Jobs(metav1.NamespaceSystem).Delete(job.Name, &metav1.DeleteOptions{PropagationPolicy: &propagation}); err != nil {
				utilruntime.HandleError(err)
				return
			}
		}
	}
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.sync(key.(string))

	c.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		glog.V(0).Infof("Error syncing %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.V(0).Infof("Dropping %q out of the backup controller queue after exceeding max retries: %v", key, err)
}

func (c *Controller) enqueue(cluster *kubermaticv1.Cluster) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(cluster)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", cluster, err))
		return
	}

	c.queue.Add(key)
}

func (c *Controller) sync(key string) error {
	clusterFromCache, err := c.clusterLister.Get(key)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("cluster '%s' in work queue no longer exists", key)
			return nil
		}
		return err
	}

	if clusterFromCache.Spec.Pause {
		glog.V(6).Infof("skipping cluster %s due to it was set to paused", key)
		return nil
	}

	if clusterFromCache.Labels[kubermaticv1.WorkerNameLabelKey] != c.workerName {
		glog.V(8).Infof("skipping cluster %s due to different worker assigned to it", key)
		return nil
	}

	cluster := clusterFromCache.DeepCopy()

	glog.V(4).Infof("syncing cluster %s", cluster.Name)

	// Cluster got deleted - regardless if the cluster was ever running, we cleanup
	if cluster.DeletionTimestamp != nil {
		// Need to cleanup
		if sets.NewString(cluster.Finalizers...).Has(cleanupFinalizer) {
			job := c.cleanupJob(cluster)
			if _, err = c.kubernetesClient.BatchV1().Jobs(metav1.NamespaceSystem).Create(job); err != nil {
				// Otherwise we end up in a loop when we are able to create the job but not remove the finalizer.
				if !kerrors.IsAlreadyExists(err) {
					return err
				}
			}

			finalizers := sets.NewString(cluster.Finalizers...)
			finalizers.Delete(cleanupFinalizer)
			cluster.Finalizers = finalizers.List()
			if cluster, err = c.kubermaticClient.KubermaticV1().Clusters().Update(cluster); err != nil {
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
		if cluster, err = c.kubermaticClient.KubermaticV1().Clusters().Update(cluster); err != nil {
			return fmt.Errorf("failed to update cluster after adding cleanup finalizer: %v", err)
		}
	}

	if err := c.ensureCronJobSecret(cluster); err != nil {
		return fmt.Errorf("failed to create backup secret: %v", err)
	}

	return c.ensureCronJob(cluster)
}

type secretData struct {
	cluster      *kubermaticv1.Cluster
	secretLister corev1lister.SecretLister
}

func (d *secretData) GetCA(name string) (*triple.KeyPair, error) {
	return resources.GetClusterCAFromLister(name, d.cluster, d.secretLister)
}

func (d *secretData) GetClusterRef() metav1.OwnerReference {
	return resources.GetClusterRef(d.cluster)
}

func (c *Controller) getEtcdSecretName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("cluster-%s-etcd-client-certificate", cluster.Name)
}

func (c *Controller) ensureCronJobSecret(cluster *kubermaticv1.Cluster) error {
	name := c.getEtcdSecretName(cluster)

	existing, err := c.secretLister.Secrets(metav1.NamespaceSystem).Get(name)
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	create := certificates.GetClientCertificateCreator(
		resources.CASecretName,
		name,
		"backup",
		nil,
		resources.BackupEtcdClientCertificateCertSecretKey,
		resources.BackupEtcdClientCertificateKeySecretKey,
	)

	data := secretData{
		cluster:      cluster,
		secretLister: c.secretLister,
	}

	se, err := create(&data, existing.DeepCopy())
	if err != nil {
		return fmt.Errorf("failed to build Secret: %v", err)
	}

	if equality.Semantic.DeepEqual(se, existing) {
		return nil
	}

	if existing == nil {
		if _, err = c.kubernetesClient.CoreV1().Secrets(metav1.NamespaceSystem).Create(se); err != nil {
			return fmt.Errorf("failed to create Secret %s: %v", se.Name, err)
		}
	} else if _, err = c.kubernetesClient.CoreV1().Secrets(metav1.NamespaceSystem).Update(se); err != nil {
		return fmt.Errorf("failed to update Secret %s: %v", se.Name, err)
	}

	return nil
}

func (c *Controller) ensureCronJob(cluster *kubermaticv1.Cluster) error {
	cronJob, err := c.cronJob(cluster)
	if err != nil {
		return err
	}

	existing, err := c.cronJobLister.CronJobs(metav1.NamespaceSystem).Get(cronJob.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		if _, err := c.kubernetesClient.BatchV1beta1().CronJobs(metav1.NamespaceSystem).Create(cronJob); err != nil {
			if !kerrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create cronjob: %v", err)
			}
		}
		c.metrics.CronJobCreationTimestamp.With(
			prometheus.Labels{"cluster": cluster.Name}).Set(float64(time.Now().UnixNano()))
		return nil
	}

	if equality.Semantic.DeepEqual(existing.Spec, cronJob.Spec) {
		return nil
	}

	if _, err := c.kubernetesClient.BatchV1beta1().CronJobs(metav1.NamespaceSystem).Update(cronJob); err != nil {
		return fmt.Errorf("failed to update cronJob: %v", err)
	}
	c.metrics.CronJobUpdateTimestamp.With(
		prometheus.Labels{"cluster": cluster.Name}).Set(float64(time.Now().UnixNano()))
	return nil
}

func (c *Controller) cleanupJob(cluster *kubermaticv1.Cluster) *v1.Job {
	cleanupContainer := c.cleanupContainer.DeepCopy()
	cleanupContainer.Env = append(cleanupContainer.Env, corev1.EnvVar{
		Name:  clusterEnvVarKey,
		Value: cluster.Name,
	})
	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("remove-cluster-backups-%s", cluster.Name),
			Labels: map[string]string{
				resources.AppLabelKey: backupCleanupJobLabel,
			},
		},
		Spec: v1.JobSpec{
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
	return job
}

func (c *Controller) cronJob(cluster *kubermaticv1.Cluster) (*batchv1beta1.CronJob, error) {
	// Name and Namespace
	cronJob := batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", cronJobPrefix, cluster.Name),
			Namespace: metav1.NamespaceSystem,
		},
	}

	// OwnerRef
	gv := kubermaticv1.SchemeGroupVersion
	cronJob.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(cluster, gv.WithKind(kubermaticv1.ClusterKindName)),
	}

	// Spec
	cronJob.Spec.Schedule = c.backupScheduleString
	cronJob.Spec.ConcurrencyPolicy = batchv1beta1.ForbidConcurrent
	cronJob.Spec.Suspend = boolPtr(false)
	cronJob.Spec.SuccessfulJobsHistoryLimit = int32Ptr(int32(0))
	etcdServiceAddr := fmt.Sprintf("https://%s.%s.svc.cluster.local.:2379", resources.EtcdClientServiceName, cluster.Status.NamespaceName)
	cronJob.Spec.JobTemplate.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:  "backup-creator",
			Image: c.backupContainerImage,
			Env: []corev1.EnvVar{
				{
					Name:  "ETCDCTL_API",
					Value: "3",
				},
			},
			Command: []string{
				"/usr/local/bin/etcdctl",
				"--endpoints", etcdServiceAddr,
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
					Name:      c.getEtcdSecretName(cluster),
					MountPath: "/etc/etcd/client",
				},
			},
		},
	}

	storeContainer := c.storeContainer.DeepCopy()
	storeContainer.Env = append(storeContainer.Env, corev1.EnvVar{
		Name:  clusterEnvVarKey,
		Value: cluster.Name,
	})
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
			Name: c.getEtcdSecretName(cluster),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  c.getEtcdSecretName(cluster),
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}

	return &cronJob, nil
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
