package cluster

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	etcdoperatorv1beta2informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/etcdoperator/v1beta2"
	kubermaticv1informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
	etcdoperatorv1beta2lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/etcdoperator/v1beta2"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	corev1informers "k8s.io/client-go/informers/core/v1"
	extensionsv1beta1informers "k8s.io/client-go/informers/extensions/v1beta1"
	rbacv1beta1informers "k8s.io/client-go/informers/rbac/v1beta1"
	"k8s.io/client-go/kubernetes"
	corev1lister "k8s.io/client-go/listers/core/v1"
	extensionsv1beta1lister "k8s.io/client-go/listers/extensions/v1beta1"
	rbacv1beta1lister "k8s.io/client-go/listers/rbac/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	validatingSyncPeriod = 15 * time.Second
	launchingSyncPeriod  = 2 * time.Second
	deletingSyncPeriod   = 10 * time.Second
	runningSyncPeriod    = 60 * time.Second
)

// GroupRunStopper represents a control loop started with Run,
// which can be terminated by closing the stop channel
type GroupRunStopper interface {
	Run(workerCount int, stop <-chan struct{})
}

type controller struct {
	kubermaticClient kubermaticclientset.Interface
	kubeClient       kubernetes.Interface

	masterResourcesPath string
	externalURL         string
	dcs                 map[string]provider.DatacenterMeta
	cps                 map[string]provider.CloudProvider

	queue      workqueue.RateLimitingInterface
	workerName string

	versions              map[string]*apiv1.MasterVersion
	updates               []apiv1.MasterUpdate
	defaultMasterVersion  *apiv1.MasterVersion
	automaticUpdateSearch *version.UpdatePathSearch

	metrics ControllerMetrics

	ClusterLister            kubermaticv1lister.ClusterLister
	EtcdClusterLister        etcdoperatorv1beta2lister.EtcdClusterLister
	NamespaceLister          corev1lister.NamespaceLister
	SecretLister             corev1lister.SecretLister
	ServiceLister            corev1lister.ServiceLister
	PvcLister                corev1lister.PersistentVolumeClaimLister
	ConfigMapLister          corev1lister.ConfigMapLister
	ServiceAccountLister     corev1lister.ServiceAccountLister
	DeploymentLister         extensionsv1beta1lister.DeploymentLister
	IngressLister            extensionsv1beta1lister.IngressLister
	ClusterRoleBindingLister rbacv1beta1lister.ClusterRoleBindingLister
}

// ControllerMetrics contains metrics about the clusters & workers
type ControllerMetrics struct {
	Clusters      metrics.Gauge
	ClusterPhases metrics.Gauge
	Workers       metrics.Gauge
}

// NewController creates a cluster controller.
func NewController(
	kubeClient kubernetes.Interface,
	kubermaticClient kubermaticclientset.Interface,
	versions map[string]*apiv1.MasterVersion,
	updates []apiv1.MasterUpdate,
	masterResourcesPath string,
	externalURL string,
	workerName string,
	dcs map[string]provider.DatacenterMeta,
	cps map[string]provider.CloudProvider,
	metrics ControllerMetrics,

	ClusterInformer kubermaticv1informers.ClusterInformer,
	EtcdClusterInformer etcdoperatorv1beta2informers.EtcdClusterInformer,
	NamespaceInformer corev1informers.NamespaceInformer,
	SecretInformer corev1informers.SecretInformer,
	ServiceInformer corev1informers.ServiceInformer,
	PvcInformer corev1informers.PersistentVolumeClaimInformer,
	ConfigMapInformer corev1informers.ConfigMapInformer,
	ServiceAccountInformer corev1informers.ServiceAccountInformer,
	DeploymentInformer extensionsv1beta1informers.DeploymentInformer,
	IngressInformer extensionsv1beta1informers.IngressInformer,
	ClusterRoleBindingInformer rbacv1beta1informers.ClusterRoleBindingInformer,
) (GroupRunStopper, error) {
	cc := &controller{
		kubermaticClient: kubermaticClient,
		kubeClient:       kubeClient,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "cluster"),

		updates:  updates,
		versions: versions,

		masterResourcesPath: masterResourcesPath,
		externalURL:         externalURL,
		workerName:          workerName,
		dcs:                 dcs,
		cps:                 cps,
		metrics:             metrics,
	}

	ClusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cc.enqueue(obj.(*kubermaticv1.Cluster))
		},
		UpdateFunc: func(old, cur interface{}) {
			cc.enqueue(cur.(*kubermaticv1.Cluster))
		},
		DeleteFunc: func(obj interface{}) {
			cluster, ok := obj.(*kubermaticv1.Cluster)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				cluster, ok = tombstone.Obj.(*kubermaticv1.Cluster)
				if !ok {
					utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a Cluster %#v", obj))
					return
				}
			}
			cc.enqueue(cluster)
		},
	})

	//In case one of our child objects change, we should update our state
	NamespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	DeploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	SecretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	ServiceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	IngressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	PvcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	ConfigMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	ServiceAccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	ClusterRoleBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	EtcdClusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})

	cc.ClusterLister = ClusterInformer.Lister()
	cc.EtcdClusterLister = EtcdClusterInformer.Lister()
	cc.NamespaceLister = NamespaceInformer.Lister()
	cc.SecretLister = SecretInformer.Lister()
	cc.ServiceLister = ServiceInformer.Lister()
	cc.PvcLister = PvcInformer.Lister()
	cc.ConfigMapLister = ConfigMapInformer.Lister()
	cc.ServiceAccountLister = ServiceAccountInformer.Lister()
	cc.DeploymentLister = DeploymentInformer.Lister()
	cc.IngressLister = IngressInformer.Lister()
	cc.ClusterRoleBindingLister = ClusterRoleBindingInformer.Lister()

	var err error
	cc.defaultMasterVersion, err = version.DefaultMasterVersion(versions)
	if err != nil {
		return nil, fmt.Errorf("could not get default master version: %v", err)
	}

	var automaticUpdates []apiv1.MasterUpdate
	for _, u := range cc.updates {
		if u.Automatic {
			automaticUpdates = append(automaticUpdates, u)
		}
	}
	cc.automaticUpdateSearch = version.NewUpdatePathSearch(cc.versions, automaticUpdates, version.SemverMatcher{})

	return cc, nil
}

func (cc *controller) enqueue(cluster *kubermaticv1.Cluster) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(cluster)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", cluster, err))
		return
	}

	cc.queue.Add(key)
}

func (cc *controller) updateCluster(originalData []byte, modifiedCluster *kubermaticv1.Cluster) error {
	currentCluster, err := cc.ClusterLister.Get(modifiedCluster.Name)
	if err != nil {
		return err
	}

	currentData, err := json.Marshal(currentCluster)
	if err != nil {
		return err
	}

	modifiedData, err := json.Marshal(modifiedCluster)
	if err != nil {
		return err
	}

	patchData, err := jsonmergepatch.CreateThreeWayJSONMergePatch(originalData, modifiedData, currentData)
	if err != nil {
		return err
	}
	//Avoid empty patch calls
	if string(patchData) == "{}" {
		return nil
	}

	_, err = cc.kubermaticClient.KubermaticV1().Clusters().Patch(modifiedCluster.Name, types.MergePatchType, patchData)
	return err
}

func (cc *controller) updateClusterError(cluster *kubermaticv1.Cluster, reason kubermaticv1.ClusterStatusError, message string, originalData []byte) error {
	if cluster.Status.ErrorReason == nil || *cluster.Status.ErrorReason == reason {
		cluster.Status.ErrorMessage = &message
		cluster.Status.ErrorReason = &reason
		return cc.updateCluster(originalData, cluster)
	}
	return nil
}

func (cc *controller) syncCluster(key string) error {
	listerCluster, err := cc.ClusterLister.Get(key)
	if err != nil {
		if kubeapierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("unable to retrieve cluster %q: %v", key, err)
	}

	cluster := listerCluster.DeepCopy()
	originalData, err := json.Marshal(cluster)
	if err != nil {
		return fmt.Errorf("failed to marshal cluster %s: %v", key, err)
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != cc.workerName {
		glog.V(8).Infof("skipping cluster %s due to different worker assigned to it", key)
		return nil
	}

	glog.V(4).Infof("syncing cluster %s", key)

	for _, phase := range kubermaticv1.ClusterPhases {
		value := 0.0
		if phase == cluster.Status.Phase {
			value = 1.0
		}
		cc.metrics.ClusterPhases.With(
			"cluster", cluster.Name,
			"phase", strings.ToLower(string(phase)),
		).Set(value)
	}

	if cluster.DeletionTimestamp != nil {
		cluster.Status.Phase = kubermaticv1.DeletingClusterStatusPhase
		if err := cc.cleanupCluster(cluster); err != nil {
			return err
		}
		return cc.updateCluster(originalData, cluster)
	}

	if cluster.Status.Phase == kubermaticv1.NoneClusterStatusPhase {
		cluster.Status.Phase = kubermaticv1.ValidatingClusterStatusPhase
	}
	var updateErr error
	if err = cc.validateCluster(cluster); err != nil {
		updateErr = cc.updateClusterError(cluster, kubermaticv1.InvalidConfigurationClusterError, err.Error(), originalData)
		if updateErr != nil {
			return fmt.Errorf("failed to set the cluster error: %v", updateErr)
		}
		return err
	}

	if cluster.Status.Phase == kubermaticv1.ValidatingClusterStatusPhase {
		cluster.Status.Phase = kubermaticv1.LaunchingClusterStatusPhase
	}
	if err := cc.reconcileCluster(cluster); err != nil {
		updateErr = cc.updateClusterError(cluster, kubermaticv1.ReconcileClusterError, err.Error(), originalData)
		if updateErr != nil {
			return fmt.Errorf("failed to set the cluster error: %v", updateErr)
		}
		return err
	}

	return cc.updateCluster(originalData, cluster)
}

func (cc *controller) runWorker() {
	for cc.processNextItem() {
	}
}

func (cc *controller) processNextItem() bool {
	key, quit := cc.queue.Get()
	if quit {
		return false
	}

	defer cc.queue.Done(key)

	err := cc.syncCluster(key.(string))

	cc.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (cc *controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		cc.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if cc.queue.NumRequeues(key) < 5 {
		glog.V(0).Infof("Error syncing cluster %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		cc.queue.AddRateLimited(key)
		return
	}

	cc.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.V(0).Infof("Dropping cluster %q out of the queue: %v", key, err)
}

func (cc *controller) syncInPhase(phase kubermaticv1.ClusterPhase) {
	clusters, err := cc.ClusterLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error listing clusters during phase sync %s: %v", phase, err))
		return
	}

	for _, c := range clusters {
		if c.Status.Phase == phase {
			cc.queue.Add(c.Name)
		}
	}
}

func (cc *controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()

	cc.metrics.Workers.Set(float64(workerCount))
	glog.Infof("Starting cluster controller with %d workers", workerCount)

	for i := 0; i < workerCount; i++ {
		go wait.Until(cc.runWorker, time.Second, stopCh)
	}

	go wait.Until(func() { cc.syncInPhase(kubermaticv1.ValidatingClusterStatusPhase) }, validatingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(kubermaticv1.LaunchingClusterStatusPhase) }, launchingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(kubermaticv1.DeletingClusterStatusPhase) }, deletingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(kubermaticv1.RunningClusterStatusPhase) }, runningSyncPeriod, stopCh)

	<-stopCh

	glog.Info("Shutting down cluster controller")
}

func (cc *controller) handleChildObject(i interface{}) {
	obj, ok := i.(metav1.Object)
	//Object might be a tombstone
	if !ok {
		tombstone, ok := i.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get obj from tombstone %#v", obj))
			return
		}
		obj = tombstone.Obj.(metav1.Object)
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(obj); controllerRef != nil {
		if controllerRef.APIVersion != kubermaticv1.SchemeGroupVersion.String() || controllerRef.Kind != "Cluster" {
			//Not for us
			return
		}
		c, err := cc.ClusterLister.Get(controllerRef.Name)
		if err != nil {
			if kubeapierrors.IsNotFound(err) {
				utilruntime.HandleError(fmt.Errorf("orphaned child obj found '%s/%s'. Responsible controller %s not found", obj.GetNamespace(), obj.GetName(), controllerRef.Name))
				return
			}
			utilruntime.HandleError(fmt.Errorf("failed to get cluster %s from lister: %v", controllerRef.Name, err))
			return
		}

		cc.enqueue(c)
		return
	}
}
