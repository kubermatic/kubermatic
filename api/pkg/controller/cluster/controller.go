package cluster

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"

	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	mastercrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned"
	seedcrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	masterinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/master"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/seed"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	workerPeriod = time.Second

	validatingSyncPeriod = 15 * time.Second
	launchingSyncPeriod  = 2 * time.Second
	deletingSyncPeriod   = 10 * time.Second
	runningSyncPeriod    = 60 * time.Second
)

// GroupRunStopper represents a control loop started with Run,
// which can be terminated by closing the stop channel
type GroupRunStopper interface {
	Run(workerCount int, stop chan struct{})
}

// SeedClientProvider offers functions to get resources of a seed-cluster
type SeedClientProvider interface {
	GetClient(dc string) (kubernetes.Interface, error)
	GetCRDClient(dc string) (seedcrdclient.Interface, error)
	GetInformerGroup(dc string) (*seed.Group, error)
}

type controller struct {
	clientProvider SeedClientProvider

	masterCrdClient       mastercrdclient.Interface
	masterResourcesPath   string
	externalURL           string
	apiserverExternalPort int
	dcs                   map[string]provider.DatacenterMeta

	queue               workqueue.RateLimitingInterface
	masterInformerGroup *masterinformer.Group

	cps        map[string]provider.CloudProvider
	workerName string

	versions              map[string]*apiv1.MasterVersion
	updates               []apiv1.MasterUpdate
	defaultMasterVersion  *apiv1.MasterVersion
	automaticUpdateSearch *version.UpdatePathSearch

	metrics ControllerMetrics
}

// ControllerMetrics contains metrics about the clusters & workers
type ControllerMetrics struct {
	Clusters metrics.Gauge
	Workers  metrics.Gauge
}

// NewController creates a cluster controller.
func NewController(
	clientProvider SeedClientProvider,
	masterCrdClient mastercrdclient.Interface,
	cps map[string]provider.CloudProvider,
	versions map[string]*apiv1.MasterVersion,
	updates []apiv1.MasterUpdate,
	masterResourcesPath string,
	externalURL string,
	workerName string,
	apiserverExternalPort int,
	dcs map[string]provider.DatacenterMeta,
	masterInformerGroup *masterinformer.Group,
	metrics ControllerMetrics,
) (GroupRunStopper, error) {
	cc := &controller{
		clientProvider:        clientProvider,
		masterCrdClient:       masterCrdClient,
		queue:                 workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "namespace"),
		cps:                   cps,
		updates:               updates,
		versions:              versions,
		masterResourcesPath:   masterResourcesPath,
		externalURL:           externalURL,
		workerName:            workerName,
		apiserverExternalPort: apiserverExternalPort,
		dcs:                 dcs,
		masterInformerGroup: masterInformerGroup,
		metrics:             metrics,
	}

	cc.masterInformerGroup.ClusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				cc.queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				cc.queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				cc.queue.Add(key)
			}
		},
	})

	// setup update controller
	var err error
	cc.defaultMasterVersion, err = version.DefaultMasterVersion(versions)
	if err != nil {
		return nil, fmt.Errorf("could not get default master version: %v", err)
	}

	automaticUpdates := []apiv1.MasterUpdate{}
	for _, u := range cc.updates {
		if u.Automatic {
			automaticUpdates = append(automaticUpdates, u)
		}
	}
	cc.automaticUpdateSearch = version.NewUpdatePathSearch(cc.versions, automaticUpdates, version.SemverMatcher{})

	return cc, nil
}

func (cc *controller) updateCluster(originalData []byte, modifiedCluster *kubermaticv1.Cluster) error {
	currentCluster, err := cc.masterInformerGroup.ClusterInformer.Lister().Get(modifiedCluster.Name)
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

	_, err = cc.masterCrdClient.KubermaticV1().Clusters().Patch(modifiedCluster.Name, types.MergePatchType, patchData)
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
	listerCluster, err := cc.masterInformerGroup.ClusterInformer.Lister().Get(key)
	if err != nil {
		if errors.IsNotFound(err) {
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
	if cluster, err = cc.validateCluster(cluster); err != nil {
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

	glog.V(4).Infof("syncing cluster %s", key)

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
	clusters, err := cc.masterInformerGroup.ClusterInformer.Lister().List(labels.Everything())
	if err != nil {
		glog.Errorf("Error listing clusters: %v", err)
	}

	for _, c := range clusters {
		if c.Status.Phase == phase {
			cc.queue.Add(c.Name)
		}
	}
}

func (cc *controller) Run(workerCount int, stopCh chan struct{}) {
	defer utilruntime.HandleCrash()

	cc.metrics.Workers.Set(float64(workerCount))
	glog.Infof("Starting cluster controller with %d workers", workerCount)

	for i := 0; i < workerCount; i++ {
		go wait.Until(cc.runWorker, workerPeriod, stopCh)
	}

	go wait.Until(func() { cc.syncInPhase(kubermaticv1.ValidatingClusterStatusPhase) }, validatingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(kubermaticv1.LaunchingClusterStatusPhase) }, launchingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(kubermaticv1.DeletingClusterStatusPhase) }, deletingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(kubermaticv1.RunningClusterStatusPhase) }, runningSyncPeriod, stopCh)

	<-stopCh

	glog.Info("Shutting down cluster controller")
}
