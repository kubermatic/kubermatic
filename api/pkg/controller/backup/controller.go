package backup

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/golang/glog"
	"github.com/robfig/cron"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informersbatchv1beta1 "k8s.io/client-go/informers/batch/v1beta1"
	"k8s.io/client-go/kubernetes"
	listersbatchv1beta1 "k8s.io/client-go/listers/batch/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	// SharedVolumeName is the name of the `emptyDir` volume the initContainer
	// will write the backup to
	SharedVolumeName = "etcd-backup"
	// DefaultBackupContainerImage holds the default Image used for creating the etcd backups
	DefaultBackupContainerImage = "quay.io/coreos/etcd:v3.3"
	// DefaultBackupInterval defines the default interval used to create backups
	DefaultBackupInterval = "20m"
	cronJobPrefix         = "etcd-backup"
)

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers                  metrics.Gauge
	CronJobCreationTimestamp metrics.Gauge
	CronJobUpdateTimestamp   metrics.Gauge
}

// Controller stores all components required to create backups
type Controller struct {
	storeContainer corev1.Container
	// backupScheduleString is the cron string representing
	// the backupSchedule
	backupScheduleString string
	// backupContainerImage holds the image used for creating the etcd backup
	// It must be configurable to cover offline use cases
	backupContainerImage string
	// workerName holds the name of this worker, only clusters with matching `.Spec.WorkerName` will be worked on
	workerName string
	metrics    Metrics

	queue            workqueue.RateLimitingInterface
	kubermaticClient kubermaticclientset.Interface
	kubernetesClient kubernetes.Interface
	clusterLister    kubermaticv1lister.ClusterLister
	cronJobLister    listersbatchv1beta1.CronJobLister
	clusterSynced    cache.InformerSynced
	cronJobSynced    cache.InformerSynced
}

// New creates a new Backup controller that is responsible for creating backupjobs
// for all managed user clusters
func New(
	storeContainer corev1.Container,
	backupSchedule time.Duration,
	backupContainerImage string,
	workerName string,
	metrics Metrics,
	kubermaticClient kubermaticclientset.Interface,
	kubernetesClient kubernetes.Interface,
	clusterInformer kubermaticv1informers.ClusterInformer,
	cronJobInformer informersbatchv1beta1.CronJobInformer) (*Controller, error) {
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
		queue:                workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Update"),
		kubermaticClient:     kubermaticClient,
		kubernetesClient:     kubernetesClient,
		backupScheduleString: backupScheduleString,
		storeContainer:       storeContainer,
		backupContainerImage: backupContainerImage,
		workerName:           workerName,
		metrics:              metrics,
	}
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
	c.clusterSynced = clusterInformer.Informer().HasSynced
	c.cronJobLister = cronJobInformer.Lister()
	c.cronJobSynced = cronJobInformer.Informer().HasSynced
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
	glog.Infof("Starting Backup controller with %d workers", workerCount)
	defer glog.Info("Shutting down Backup  controller")

	if !cache.WaitForCacheSync(stopCh, c.clusterSynced, c.cronJobSynced) {
		runtime.HandleError(errors.New("unable to sync caches for Backup controller"))
		return
	}

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	c.metrics.Workers.Set(float64(workerCount))
	<-stopCh
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
	glog.V(2).Infof("Processing cluster %s", clusterFromCache.Name)

	if clusterFromCache.Spec.WorkerName != c.workerName {
		return nil
	}
	// We need to know the namespace otherwise we do not know
	// the address of the etcd
	if clusterFromCache.Status.NamespaceName == "" {
		return nil
	}

	cluster := clusterFromCache.DeepCopy()

	cronJob, err := c.cronJob(cluster)
	if err != nil {
		return err
	}

	existing, err := c.cronJobLister.CronJobs(metav1.NamespaceSystem).Get(fmt.Sprintf("%s-%s", cronJobPrefix, cluster.Name))
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		if _, err := c.kubernetesClient.BatchV1beta1().CronJobs(metav1.NamespaceSystem).Create(cronJob); err != nil {
			return fmt.Errorf("failed to create cronjob: %v", err)
		}
		c.metrics.CronJobCreationTimestamp.With("cluster", cluster.Name).Set(float64(time.Now().UnixNano()))
		return nil
	}

	if equal := apiequality.Semantic.DeepEqual(existing.Spec, cronJob.Spec); equal {
		return nil
	}

	if _, err := c.kubernetesClient.BatchV1beta1().CronJobs(metav1.NamespaceSystem).Update(cronJob); err != nil {
		return fmt.Errorf("failed to update cronJob: %v", err)
	}
	c.metrics.CronJobUpdateTimestamp.With("cluster", cluster.Name).Set(float64(time.Now().UnixNano()))
	return nil
}

func (c *Controller) cronJob(cluster *kubermaticv1.Cluster) (*batchv1beta1.CronJob, error) {
	// Name and Namespace
	cronJob := batchv1beta1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", cronJobPrefix, cluster.Name),
		Namespace: metav1.NamespaceSystem}}

	// OwnerRef
	gv := kubermaticv1.SchemeGroupVersion
	cronJob.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(cluster,
		gv.WithKind(kubermaticv1.ClusterKindName))}

	// Spec
	cronJob.Spec.Schedule = c.backupScheduleString
	cronJob.Spec.ConcurrencyPolicy = batchv1beta1.ForbidConcurrent
	cronJob.Spec.Suspend = boolPtr(false)
	etcdServiceAddr := fmt.Sprintf("etcd.%s.cluster.local.:2379", cluster.Status.NamespaceName)
	cronJob.Spec.JobTemplate.Spec.Template.Spec.InitContainers = []corev1.Container{
		corev1.Container{Name: "backup-creator",
			Image:        c.backupContainerImage,
			Env:          []corev1.EnvVar{corev1.EnvVar{Name: "ETCDCTL_API", Value: "3"}},
			Command:      []string{"/usr/local/bin/etcdctl", "--endpoints", etcdServiceAddr, "snapshot", "save", "/backup/snap.db"},
			VolumeMounts: []corev1.VolumeMount{corev1.VolumeMount{Name: SharedVolumeName, MountPath: "/backup"}}},
	}
	cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{c.storeContainer}
	cronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: SharedVolumeName,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
	cronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure

	return &cronJob, nil
}

func boolPtr(b bool) *bool {
	return &b
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
