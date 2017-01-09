package cluster

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/addons/manager"
	"github.com/kubermatic/api/controller"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider"
	kprovider "github.com/kubermatic/api/provider/kubernetes"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kerrors "k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/types"
	uruntime "k8s.io/client-go/pkg/util/runtime"
	"k8s.io/client-go/pkg/util/wait"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

const (
	fullResyncPeriod               = 5 * time.Minute
	namespaceStoreSyncedPollPeriod = 100 * time.Millisecond
	workerNum                      = 5
	maxUpdateRetries               = 5
	launchTimeout                  = 5 * time.Minute

	maxAddonInstallAttempts   = 3
	waitBetweenInstallAttempt = 30 * time.Second

	workerPeriod        = time.Second
	queueResyncPeriod   = 10 * time.Millisecond
	pendingSyncPeriod   = 10 * time.Second
	launchingSyncPeriod = 2 * time.Second
	runningSyncPeriod   = 1 * time.Minute
)

type clusterController struct {
	dc                  string
	tprClient           extensions.Clientset
	client              kubernetes.Interface
	queue               *cache.FIFO // of namespace keys
	recorder            record.EventRecorder
	masterResourcesPath string
	externalURL         string
	overwriteHost       string
	addonResourcesPath  string
	// store namespaces with the role=kubermatic-cluster label
	nsController *cache.Controller
	nsStore      cache.Store

	podController *cache.Controller
	podStore      cache.StoreToPodLister

	depController *cache.Controller
	depStore      cache.Indexer

	secretController *cache.Controller
	secretStore      cache.Indexer

	serviceController *cache.Controller
	serviceStore      cache.Indexer

	ingressController *cache.Controller
	ingressStore      cache.Indexer

	addonController *cache.Controller
	addonStore      cache.Store

	pvcController *cache.Controller
	pvcStore      cache.Indexer

	// non-thread safe:
	mu         sync.Mutex
	cps        map[string]provider.CloudProvider
	inProgress map[string]struct{} // in progress clusters
	dev        bool
}

// NewController creates a cluster controller.
func NewController(
	dc string,
	client kubernetes.Interface,
	tprClient extensions.Clientset,
	cps map[string]provider.CloudProvider,
	masterResourcesPath string,
	externalURL string,
	dev bool,
	overwriteHost string,
	addonResourcesPath string,
) (controller.Controller, error) {
	cc := &clusterController{
		dc:                  dc,
		client:              client,
		tprClient:           tprClient,
		queue:               cache.NewFIFO(func(obj interface{}) (string, error) { return obj.(string), nil }),
		cps:                 cps,
		inProgress:          map[string]struct{}{},
		masterResourcesPath: masterResourcesPath,
		externalURL:         externalURL,
		dev:                 dev,
		overwriteHost:       overwriteHost,
		addonResourcesPath:  addonResourcesPath,
	}

	eventBroadcaster := record.NewBroadcaster()
	cc.recorder = eventBroadcaster.NewRecorder(v1.EventSource{Component: "clustermanager"})
	eventBroadcaster.StartLogging(glog.Infof)
	e := cc.client.CoreV1().Events("")
	es := corev1.EventSinkImpl{Interface: e}
	eventBroadcaster.StartRecordingToSink(&es)

	nsLabels := map[string]string{
		kprovider.RoleLabelKey: kprovider.ClusterRoleLabel,
	}
	if dev {
		nsLabels[kprovider.DevLabelKey] = kprovider.DevLabelValue
	}

	cc.nsStore, cc.nsController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = labels.SelectorFromSet(labels.Set(nsLabels)).String()
				return cc.client.CoreV1().Namespaces().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return cc.client.CoreV1().Namespaces().Watch(options)
			},
		},
		&v1.Namespace{},
		fullResyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ns := obj.(*v1.Namespace)
				glog.V(4).Infof("Adding cluster %q", ns.Name)
				cc.enqueue(ns)
			},
			UpdateFunc: func(old, cur interface{}) {
				ns := cur.(*v1.Namespace)
				glog.V(4).Infof("Updating cluster %q", ns.Name)
				cc.enqueue(ns)
			},
			DeleteFunc: func(obj interface{}) {
				ns := obj.(*v1.Namespace)
				glog.V(4).Infof("Deleting cluster %q", ns.Name)
				cc.enqueue(ns)
			},
		},
	)

	namespaceIndexer := cache.Indexers{
		"namespace": cache.IndexFunc(cache.MetaNamespaceIndexFunc),
	}

	cc.podStore.Indexer, cc.podController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return cc.client.CoreV1().Pods(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return cc.client.CoreV1().Pods(v1.NamespaceAll).Watch(options)
			},
		},
		&v1.Pod{},
		fullResyncPeriod,
		cache.ResourceEventHandlerFuncs{},
		namespaceIndexer,
	)

	cc.depStore, cc.depController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return cc.client.ExtensionsV1beta1().Deployments(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return cc.client.ExtensionsV1beta1().Deployments(v1.NamespaceAll).Watch(options)
			},
		},
		&v1beta1.Deployment{},
		fullResyncPeriod,
		cache.ResourceEventHandlerFuncs{},
		namespaceIndexer,
	)

	cc.secretStore, cc.secretController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return cc.client.CoreV1().Secrets(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return cc.client.CoreV1().Secrets(v1.NamespaceAll).Watch(options)
			},
		},
		&v1.Secret{},
		fullResyncPeriod,
		cache.ResourceEventHandlerFuncs{},
		namespaceIndexer,
	)

	cc.serviceStore, cc.serviceController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return cc.client.CoreV1().Services(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return cc.client.CoreV1().Services(v1.NamespaceAll).Watch(options)
			},
		},
		&v1.Service{},
		fullResyncPeriod,
		cache.ResourceEventHandlerFuncs{},
		namespaceIndexer,
	)

	cc.ingressStore, cc.ingressController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return cc.client.ExtensionsV1beta1().Ingresses(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return cc.client.ExtensionsV1beta1().Ingresses(v1.NamespaceAll).Watch(options)
			},
		},
		&v1beta1.Ingress{},
		fullResyncPeriod,
		cache.ResourceEventHandlerFuncs{},
		namespaceIndexer,
	)

	cc.pvcStore, cc.pvcController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return cc.client.CoreV1().PersistentVolumeClaims(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return cc.client.CoreV1().PersistentVolumeClaims(v1.NamespaceAll).Watch(options)
			},
		},
		&v1.PersistentVolumeClaim{},
		fullResyncPeriod,
		cache.ResourceEventHandlerFuncs{},
		namespaceIndexer,
	)

	cc.addonStore, cc.addonController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return cc.tprClient.ClusterAddons(v1.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return cc.tprClient.ClusterAddons(v1.NamespaceAll).Watch(options)
			},
		},
		&extensions.ClusterAddon{},
		1*time.Minute,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				addon := obj.(*extensions.ClusterAddon)
				cc.syncAddon(addon)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newAddon := newObj.(*extensions.ClusterAddon)
				glog.V(8).Infof("Detected update on addon %s/%s", newAddon.Metadata.Namespace, newAddon.Metadata.Name)
				cc.syncAddon(newAddon)
			},
		},
	)

	return cc, nil
}

func (cc *clusterController) syncAddon(a *extensions.ClusterAddon) {
	storeAddon, exists, err := cc.addonStore.GetByKey(fmt.Sprintf("%s/%s", a.Metadata.Namespace, a.Metadata.Name))
	if !exists || err != nil || storeAddon == nil {
		glog.Errorf("could not fetch addon %s from store: %v", a.Metadata.Name, err)
		return
	}

	addon := storeAddon.(*extensions.ClusterAddon)
	if addon.Phase != extensions.PendingAddonStatusPhase {
		return
	}

	obj, exists, err := cc.nsStore.GetByKey(a.Metadata.Namespace)
	if !exists {
		glog.Errorf("Namespace for cluster %s does not exist", a.Metadata.Namespace)
		return
	}
	if err != nil {
		glog.Errorf("failed to get namespace for %s: : %v", a.Metadata.Namespace, err)
		return
	}
	ns := obj.(*v1.Namespace)
	cluster, err := kprovider.UnmarshalCluster(cc.cps, ns)
	if err != nil {
		glog.Errorf("failed to unmarshal cluster(ns) %s during addon-install: %v", a.Metadata.Namespace, err)
		return
	}

	if cluster.Spec.Dev != cc.dev {
		return
	}

	if cluster.Status.Phase != api.RunningClusterStatusPhase {
		glog.Infof("Postponed addon install. cluster %s is not ready", a.Metadata.Namespace)
		return
	}

	addon.Phase = extensions.InstallingAddonStatusPhase
	addon, err = cc.tprClient.ClusterAddons(addon.Metadata.GetNamespace()).Update(addon)
	if err != nil {
		glog.Error(err)
	}
	defer func() {
		//Release lock in case something bad happened
		if addon.Phase == extensions.InstallingAddonStatusPhase {
			addon.Phase = extensions.PendingAddonStatusPhase
			addon, err = cc.tprClient.ClusterAddons(addon.Metadata.GetNamespace()).Update(addon)
			if err != nil {
				glog.Error(err)
			}
		}
	}()

	glog.V(4).Infof("Installing addon %s", addon.Name)

	addonManager, err := manager.NewHelmAddonManager(cluster.GetKubeconfig(), cc.addonResourcesPath)
	if err != nil {
		glog.Errorf("failed to create addonManager:%s", addon.Metadata.Namespace, err)
		return
	}

	installedAddon, err := addonManager.Install(addon)
	if err != nil {
		glog.Errorf("failed to install plugin %s for cluster %s: %v", addon.Name, cluster.Metadata.Name, err)
		addon.Attempt++
		if addon.Attempt >= maxAddonInstallAttempts {
			glog.Errorf("failed to install plugin after %d attempts: %v - wont try again", err)
			addon.Phase = extensions.FailedAddonStatusPhase
		} else {
			time.Sleep(waitBetweenInstallAttempt)
			addon.Phase = extensions.PendingAddonStatusPhase

		}
	} else {
		addon = installedAddon
		addon.Phase = extensions.RunningAddonStatusPhase
	}

	addon, err = cc.tprClient.ClusterAddons(addon.Metadata.GetNamespace()).Update(addon)
	if err != nil {
		glog.Error(err)
	}
}
func (cc *clusterController) recordClusterPhaseChange(ns *v1.Namespace, newPhase api.ClusterPhase) {
	ref := &v1.ObjectReference{
		Kind:      "Namespace",
		Name:      ns.Name,
		UID:       types.UID(ns.Name),
		Namespace: ns.Name,
	}
	glog.V(2).Infof("Recording phase change %s event message for namespace %s", string(newPhase), ns.Name)
	cc.recorder.Eventf(ref, v1.EventTypeNormal, string(newPhase), "Cluster phase is now: %s", newPhase)
}

func (cc *clusterController) recordClusterEvent(c *api.Cluster, reason, msg string, args ...interface{}) {
	nsName := kprovider.NamespaceName(c.Metadata.User, c.Metadata.Name)
	ref := &v1.ObjectReference{
		Kind:      "Namespace",
		Name:      nsName,
		UID:       types.UID(nsName),
		Namespace: nsName,
	}
	glog.V(4).Infof("Recording event for namespace %q: %s", nsName, fmt.Sprintf(msg, args...))
	cc.recorder.Eventf(ref, v1.EventTypeNormal, reason, msg, args)
}

func (cc *clusterController) updateCluster(oldC, newC *api.Cluster) error {
	ns := kprovider.NamespaceName(newC.Metadata.User, newC.Metadata.Name)
	for i := 0; i < maxUpdateRetries; i++ {
		// try to get current namespace
		oldNS, err := cc.client.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// update with latest cluster state
		newNS, err := func() (*v1.Namespace, error) {
			cc.mu.Lock()
			defer cc.mu.Unlock()
			return kprovider.MarshalCluster(cc.cps, newC, oldNS)
		}()
		if err != nil {
			return err
		}

		// try to write back namespace
		_, err = cc.client.CoreV1().Namespaces().Update(newNS)
		if err != nil {
			if !kerrors.IsConflict(err) {
				glog.V(4).Infof("Write conflict of namespace %q (retry=%i/%i)", ns, i, maxUpdateRetries)
				continue
			}
			return err
		}

		// record phase change events
		if oldC.Status.Phase != newC.Status.Phase {
			cc.recordClusterPhaseChange(newNS, newC.Status.Phase)
		}

		return nil
	}

	return fmt.Errorf("Updading namespace %q failed after %v retries", ns, maxUpdateRetries)
}

func (cc *clusterController) checkTimeout(c *api.Cluster) (*api.Cluster, error) {
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

func (cc *clusterController) syncClusterNamespace(key string) error {
	// only run one syncCluster for each cluster in parallel
	cc.mu.Lock()
	if _, found := cc.inProgress[key]; found {
		cc.mu.Unlock()
		glog.V(4).Infof("Skipped in-progress namespace %q", key)
		return nil
	}
	cc.inProgress[key] = struct{}{}
	defer func() {
		cc.mu.Lock()
		delete(cc.inProgress, key)
		cc.mu.Unlock()
	}()
	cc.mu.Unlock()

	// get namespace
	startTime := time.Now()
	glog.V(4).Infof("Syncing cluster %q", key)
	defer func() {
		glog.V(4).Infof("Finished syncing namespace %q (%v)", key, time.Since(startTime))
	}()
	obj, exists, err := cc.nsStore.GetByKey(key)
	if err != nil {
		glog.Infof("Unable to retrieve namespace %q from store: %v", key, err)
		addErr := cc.queue.Add(key)
		if addErr != nil {
			glog.Infof("Unable to add namespace %q to queue: %v", key, addErr)
		}
		return err
	}
	if !exists {
		glog.V(3).Infof("Namespace %q has been deleted", key)
		return nil
	}
	ns := obj.(*v1.Namespace)
	if !cc.controllersHaveSynced() {
		// Sleep so we give the pod reflector goroutine a chance to run.
		time.Sleep(namespaceStoreSyncedPollPeriod)
		glog.Infof("Waiting for controllers to sync, requeuing namespace %q", ns.Name)
		cc.enqueue(ns)
		return nil
	}

	// sync cluster
	c, err := func() (*api.Cluster, error) {
		cc.mu.Lock()
		defer cc.mu.Unlock()
		return kprovider.UnmarshalCluster(cc.cps, ns)
	}()
	if err != nil {
		return err
	}

	// state machine
	var changedC *api.Cluster
	switch c.Status.Phase {
	case api.PendingClusterStatusPhase:
		changedC, err = cc.syncPendingCluster(c)
	case api.LaunchingClusterStatusPhase:
		changedC, err = cc.syncLaunchingCluster(c)
	case api.RunningClusterStatusPhase:
		changedC, err = cc.syncRunningCluster(c)
	default:
		glog.V(5).Infof("Ignoring cluster %q in phase %q", c.Metadata.Name, c.Status.Phase)
	}
	if err != nil {
		return fmt.Errorf(
			"error in phase %q sync function of cluster %q: %v",
			c.Status.Phase,
			c.Metadata.Name,
			err,
		)
	}

	// sync back to namespace if c was changed
	if changedC != nil {
		glog.V(5).Infof(
			"Cluster %q changed in phase %q, updating namespace.",
			c.Metadata.Name,
			c.Status.Phase,
		)
		err = cc.updateCluster(c, changedC)
		if err != nil {
			return fmt.Errorf(
				"failed to update changed namespace for cluster %q: %v",
				c.Metadata.Name,
				err,
			)
		}
	}

	return nil
}

func (cc *clusterController) enqueue(ns *v1.Namespace) {
	if !cc.dev && ns.Labels[kprovider.DevLabelKey] == kprovider.DevLabelValue {
		glog.V(5).Infof("Skipping dev cluster %q", ns.Name)
		return
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(ns)
	if err != nil {
		glog.Errorf("Couldn't get key for object %+v: %v", ns, err)
		return
	}

	err = cc.queue.Add(key)
	if err != nil {
		glog.Errorf("Unable to add namespace %q to queue: %v", key, err)
		return
	}
}

func (cc *clusterController) worker() {
	nsKey, err := cc.queue.Pop(func(nsKey interface{}) error {
		return nil
	})

	if err != nil {
		glog.Errorf("Worker failed to proccess key %s: %v - will be requeued", nsKey.(string), err)
	} else {
		glog.V(4).Infof("Worker proccessed key %s", nsKey.(string))
	}

	err = cc.syncClusterNamespace(nsKey.(string))
	if err != nil {
		glog.Errorf("Error syncing cluster with key %s: %v", nsKey.(string), err)
	}

}

func (cc *clusterController) syncInPhase(phase api.ClusterPhase) {
	for _, obj := range cc.nsStore.List() {
		ns := obj.(*v1.Namespace)
		if v, found := ns.Labels[kprovider.RoleLabelKey]; !found || v != kprovider.ClusterRoleLabel {
			continue
		}
		if kprovider.ClusterPhase(ns) == phase {
			cc.enqueue(ns)
		}
	}
}

func (cc *clusterController) Run(stopCh <-chan struct{}) {
	defer uruntime.HandleCrash()
	glog.Info("Starting cluster controller")

	go cc.nsController.Run(wait.NeverStop)
	go cc.podController.Run(wait.NeverStop)
	go cc.depController.Run(wait.NeverStop)
	go cc.secretController.Run(wait.NeverStop)
	go cc.serviceController.Run(wait.NeverStop)
	go cc.ingressController.Run(wait.NeverStop)
	go cc.addonController.Run(wait.NeverStop)
	go cc.pvcController.Run(wait.NeverStop)

	for i := 0; i < workerNum; i++ {
		go wait.Until(cc.worker, workerPeriod, stopCh)
	}

	go wait.Until(func() { cc.syncInPhase(api.PendingClusterStatusPhase) }, pendingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(api.LaunchingClusterStatusPhase) }, launchingSyncPeriod, stopCh)
	go wait.Until(func() { cc.syncInPhase(api.RunningClusterStatusPhase) }, runningSyncPeriod, stopCh)
	go wait.Until(func() {
		err := cc.queue.Resync()
		if err != nil {
			glog.Errorf("Error syncing queue: %v", err)
		}
	}, queueResyncPeriod, stopCh)

	<-stopCh

	glog.Info("Shutting down cluster controller")
}

func (cc *clusterController) controllersHaveSynced() bool {
	return cc.nsController.HasSynced() &&
		cc.podController.HasSynced() &&
		cc.secretController.HasSynced() &&
		cc.depController.HasSynced() &&
		cc.serviceController.HasSynced() &&
		cc.ingressController.HasSynced() &&
		cc.addonController.HasSynced() &&
		cc.pvcController.HasSynced()
}
