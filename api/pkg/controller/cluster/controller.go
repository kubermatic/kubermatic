package cluster

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/controller/update"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	mastercrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned"
	seedcrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	masterinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/master"
	seedinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/seed"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"gopkg.in/square/go-jose.v2/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	launchTimeout = 5 * time.Minute

	workerPeriod        = time.Second
	pendingSyncPeriod   = 10 * time.Second
	launchingSyncPeriod = 2 * time.Second
	runningSyncPeriod   = 1 * time.Minute
	updatingSyncPeriod  = 5 * time.Second
	timeoutSyncPeriod   = 10 * time.Second
)

// GroupRunStopper represents a control loop started with Run,
// which can be terminated by closing the stop channel
type GroupRunStopper interface {
	Run(workerCount int, stop chan struct{})
}

type controller struct {
	dc                    string
	client                kubernetes.Interface
	seedCrdClient         seedcrdclient.Interface
	masterCrdClient       mastercrdclient.Interface
	masterResourcesPath   string
	externalURL           string
	apiserverExternalPort int
	dcs                   map[string]provider.DatacenterMeta

	queue               workqueue.RateLimitingInterface
	seedInformerGroup   *seedinformer.Group
	masterInformerGroup *masterinformer.Group

	cps        map[string]provider.CloudProvider
	workerName string

	updateController      update.Interface
	versions              map[string]*api.MasterVersion
	updates               []api.MasterUpdate
	defaultMasterVersion  *api.MasterVersion
	automaticUpdateSearch *version.UpdatePathSearch
}

// NewController creates a cluster controller.
func NewController(
	dc string,
	client kubernetes.Interface,
	seedCrdClient seedcrdclient.Interface,
	masterCrdClient mastercrdclient.Interface,
	cps map[string]provider.CloudProvider,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
	masterResourcesPath string,
	externalURL string,
	workerName string,
	apiserverExternalPort int,
	dcs map[string]provider.DatacenterMeta,
	masterInformerGroup *masterinformer.Group,
	seedInformerGroup *seedinformer.Group,
) (GroupRunStopper, error) {
	cc := &controller{
		dc:                    dc,
		client:                client,
		seedCrdClient:         seedCrdClient,
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
		seedInformerGroup:   seedInformerGroup,
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
	cc.updateController = update.New(
		cc.client,
		cc.seedCrdClient,
		cc.masterResourcesPath,
		cc.dc,
		cc.versions,
		cc.updates,
		cc.seedInformerGroup,
	)
	automaticUpdates := []api.MasterUpdate{}
	for _, u := range cc.updates {
		if u.Automatic {
			automaticUpdates = append(automaticUpdates, u)
		}
	}
	cc.automaticUpdateSearch = version.NewUpdatePathSearch(cc.versions, automaticUpdates, version.EqualityMatcher{})

	return cc, nil
}

func (cc *controller) updateCluster(originalData []byte, c *kubermaticv1.Cluster) error {
	modifiedData, err := json.Marshal(c)
	if err != nil {
		return err
	}

	patchData, err := jsonmergepatch.CreateThreeWayJSONMergePatch(nil, modifiedData, originalData)
	if err != nil {
		return err
	}
	//Avoid empty patch calls
	if string(patchData) == "{}" {
		return nil
	}

	_, err = cc.masterCrdClient.KubermaticV1().Clusters().Patch(c.Name, types.MergePatchType, patchData)
	return err
}

func (cc *controller) timeoutWorker() {
	clusters, err := cc.masterInformerGroup.ClusterInformer.Lister().List(labels.Everything())
	if err != nil {
		glog.Errorf("failed to get cluster list: %v", err)
	}

	for _, cluster := range clusters {
		if cluster.Status.Phase != kubermaticv1.LaunchingClusterStatusPhase {
			continue
		}
		now := metav1.Now()
		sinceSinceLaunching := now.Sub(cluster.Status.LastTransitionTime.Time)
		if sinceSinceLaunching > launchTimeout {
			originalData, err := json.Marshal(cluster)
			if err != nil {
				glog.Errorf("failed to marshal cluster %s: %v", cluster.Name, err)
				continue
			}
			glog.Infof("Launch timeout for cluster %q after %v", cluster.Name, sinceSinceLaunching)
			cluster.Status.Phase = kubermaticv1.FailedClusterStatusPhase
			cluster.Status.LastTransitionTime = now
			if err := cc.updateCluster(originalData, cluster); err != nil {
				glog.Errorf("failed to update failed cluster %q: %v", cluster.Name, err)
			}
		}
	}
}

func (cc *controller) syncCluster(key string) error {
	cluster, err := cc.masterCrdClient.KubermaticV1().Clusters().Get(key, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to retrieve cluster %q: %v", key, err)
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != cc.workerName {
		glog.V(8).Infof("skipping cluster %s due to different worker assigned to it", key)
		return nil
	}

	originalData, err := json.Marshal(cluster)
	if err != nil {
		return fmt.Errorf("failed to marshal cluster %s: %v", key, err)
	}

	glog.V(4).Infof("Syncing cluster %q in phase %q", cluster.Name, cluster.Status.Phase)

	// state machine
	var changedC *kubermaticv1.Cluster
	switch cluster.Status.Phase {
	case kubermaticv1.PendingClusterStatusPhase:
		changedC, err = cc.syncPendingCluster(cluster)
	case kubermaticv1.LaunchingClusterStatusPhase:
		changedC, err = cc.syncLaunchingCluster(cluster)
	case kubermaticv1.RunningClusterStatusPhase:
		changedC, err = cc.syncRunningCluster(cluster)
	case kubermaticv1.UpdatingMasterClusterStatusPhase:
		changedC, err = cc.syncUpdatingClusterMaster(cluster)
	case kubermaticv1.DeletingClusterStatusPhase:
		changedC, err = nil, nil
	default:
		return fmt.Errorf("invalid phase %q", cluster.Status.Phase)
	}
	if err != nil {
		return fmt.Errorf("error syncing cluster in phase %q: %v", cluster.Status.Phase, err)
	}

	// sync back to namespace if cc was changed
	if changedC != nil {
		err = cc.updateCluster(originalData, changedC)
		if err != nil {
			return fmt.Errorf("failed to update changed cluster %q: %v", cluster.Name, err)
		}
	}

	return nil
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
	glog.V(0).Infof("Dropping node %q out of the queue: %v", key, err)
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
	glog.Infof("Starting cluster controller with %d workers", workerCount)

	for i := 0; i < workerCount; i++ {
		go wait.Until(cc.runWorker, workerPeriod, stopCh)
	}

	go wait.Until(func() { cc.syncInPhase(kubermaticv1.PendingClusterStatusPhase) }, pendingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(kubermaticv1.LaunchingClusterStatusPhase) }, launchingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(kubermaticv1.RunningClusterStatusPhase) }, runningSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(kubermaticv1.UpdatingMasterClusterStatusPhase) }, updatingSyncPeriod, stopCh)
	go wait.Until(cc.timeoutWorker, timeoutSyncPeriod, stopCh)

	<-stopCh

	glog.Info("Shutting down cluster controller")
}
