package cluster

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/controller/update"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	crdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	seedinformer "github.com/kubermatic/kubermatic/api/pkg/kubernetes/informer/seed"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"gopkg.in/square/go-jose.v2/json"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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
)

// GroupRunStopper represents a control loop started with Run,
// which can be terminated by closing the stop channel
type GroupRunStopper interface {
	Run(workerCount int, stop chan struct{})
}

type controller struct {
	dc                    string
	client                kubernetes.Interface
	crdClient             crdclient.Interface
	masterResourcesPath   string
	externalURL           string
	apiserverExternalPort int
	dcs                   map[string]provider.DatacenterMeta

	queue             workqueue.RateLimitingInterface
	seedInformerGroup *seedinformer.Group

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
	crdClient crdclient.Interface,
	cps map[string]provider.CloudProvider,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
	masterResourcesPath string,
	externalURL string,
	workerName string,
	apiserverExternalPort int,
	dcs map[string]provider.DatacenterMeta,
	seedInformerGroup *seedinformer.Group,
) (GroupRunStopper, error) {
	cc := &controller{
		dc:                    dc,
		client:                client,
		crdClient:             crdClient,
		queue:                 workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "nodeset"),
		cps:                   cps,
		updates:               updates,
		versions:              versions,
		masterResourcesPath:   masterResourcesPath,
		externalURL:           externalURL,
		workerName:            workerName,
		apiserverExternalPort: apiserverExternalPort,
		dcs:               dcs,
		seedInformerGroup: seedInformerGroup,
	}

	cc.seedInformerGroup.NamespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
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
		cc.crdClient,
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

func (cc *controller) updateNamespace(originalData []byte, ns *v1.Namespace) error {
	modifiedData, err := json.Marshal(ns)
	if err != nil {
		return err
	}

	patchData, err := strategicpatch.CreateTwoWayMergePatch(originalData, modifiedData, v1.Namespace{})
	if err != nil {
		return err
	}
	//Avoid empty patch calls
	if string(patchData) == "{}" {
		return nil
	}

	_, err = cc.client.CoreV1().Namespaces().Patch(ns.Name, types.StrategicMergePatchType, patchData)
	return err
}

func (cc *controller) checkTimeout(c *api.Cluster) (*api.Cluster, error) {
	now := time.Now()
	timeSinceCreation := now.Sub(c.Status.LastTransitionTime)
	if timeSinceCreation > launchTimeout {
		glog.Infof("Launch timeout for cluster %q after %v", c.Metadata.Name, timeSinceCreation)
		c.Status.Phase = api.FailedClusterStatusPhase
		c.Status.LastTransitionTime = now
		return c, nil
	}

	return nil, nil
}

func (cc *controller) syncCluster(key string) error {
	ns, err := cc.client.CoreV1().Namespaces().Get(key, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to retrieve namespace %q from lister: %v", key, err)
	}

	if ns.Labels[kprovider.RoleLabelKey] != kprovider.ClusterRoleLabel {
		return nil
	}
	if ns.Labels[kprovider.WorkerNameLabelKey] != cc.workerName {
		return nil
	}

	originalData, err := json.Marshal(ns)
	if err != nil {
		return fmt.Errorf("failed to marshal namespace %s: %v", key, err)
	}

	cluster, err := kprovider.UnmarshalCluster(cc.cps, ns)
	if err != nil {
		return err
	}
	glog.V(4).Infof("Syncing cluster %q in phase %q", cluster.Metadata.Name, cluster.Status.Phase)

	// state machine
	var changedC *api.Cluster
	switch cluster.Status.Phase {
	case api.PendingClusterStatusPhase:
		changedC, err = cc.syncPendingCluster(cluster)
	case api.LaunchingClusterStatusPhase:
		changedC, err = cc.syncLaunchingCluster(cluster)
	case api.RunningClusterStatusPhase:
		changedC, err = cc.syncRunningCluster(cluster)
	case api.UpdatingMasterClusterStatusPhase:
		changedC, err = cc.syncUpdatingClusterMaster(cluster)
	case api.DeletingClusterStatusPhase:
		changedC, err = nil, nil
	default:
		return fmt.Errorf("invalid phase %q", cluster.Status.Phase)
	}
	if err != nil {
		return fmt.Errorf("error syncing cluster in phase %q: %v", cluster.Status.Phase, err)
	}

	// sync back to namespace if cc was changed
	if changedC != nil {
		ns, err = kprovider.MarshalCluster(cc.cps, changedC, ns)
		if err != nil {
			return fmt.Errorf("failed to marshal cluster %s: %v", changedC.Metadata.Name, err)
		}
		err = cc.updateNamespace(originalData, ns)
		if err != nil {
			return fmt.Errorf("failed to update changed namespace for cluster %q: %v", cluster.Metadata.Name, err)
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

func (cc *controller) syncInPhase(phase api.ClusterPhase) {
	namespaces, err := cc.seedInformerGroup.NamespaceInformer.Lister().List(labels.Everything())
	if err != nil {
		glog.Errorf("Error listing namespaces: %v", err)
	}

	for _, ns := range namespaces {
		if v, found := ns.Labels[kprovider.RoleLabelKey]; !found || v != kprovider.ClusterRoleLabel {
			continue
		}
		if kprovider.ClusterPhase(ns) == phase {
			cc.queue.Add(ns.Name)
		}
	}
}

func (cc *controller) Run(workerCount int, stopCh chan struct{}) {
	defer utilruntime.HandleCrash()
	glog.Info("Starting cluster controller")

	for i := 0; i < workerCount; i++ {
		go wait.Until(cc.runWorker, workerPeriod, stopCh)
	}

	go wait.Until(func() { cc.syncInPhase(api.PendingClusterStatusPhase) }, pendingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(api.LaunchingClusterStatusPhase) }, launchingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(api.RunningClusterStatusPhase) }, runningSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(api.UpdatingMasterClusterStatusPhase) }, updatingSyncPeriod, stopCh)

	<-stopCh

	glog.Info("Shutting down cluster controller")
}
