package monitoring

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	kubermaticv1informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsv1informer "k8s.io/client-go/informers/apps/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	rbacv1informer "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	appsv1lister "k8s.io/client-go/listers/apps/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	rbacb1lister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	// The monitoring controller waits for the cluster to become healthy,
	// before adding the monitoring components to the clusters
	healthCheckPeriod = 5 * time.Second
)

// userClusterConnectionProvider offers functions to retrieve clients for the given user clusters
type userClusterConnectionProvider interface {
	GetClient(*kubermaticv1.Cluster) (kubernetes.Interface, error)
}

// Controller stores all components required for monitoring
type Controller struct {
	kubeClient              kubernetes.Interface
	userClusterConnProvider userClusterConnectionProvider

	dcs                                              map[string]provider.DatacenterMeta
	dc                                               string
	overwriteRegistry                                string
	nodePortRange                                    string
	nodeAccessNetwork                                string
	etcdDiskSize                                     resource.Quantity
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
	dockerPullConfigJSON                             []byte

	queue workqueue.RateLimitingInterface

	clusterLister            kubermaticv1lister.ClusterLister
	serviceAccountLister     corev1lister.ServiceAccountLister
	configMapLister          corev1lister.ConfigMapLister
	roleLister               rbacb1lister.RoleLister
	roleBindingLister        rbacb1lister.RoleBindingLister
	serviceLister            corev1lister.ServiceLister
	statefulSetLister        appsv1lister.StatefulSetLister
	clusterRoleBindingLister rbacb1lister.ClusterRoleBindingLister
	deploymentLister         appsv1lister.DeploymentLister
	secretLister             corev1lister.SecretLister
}

// New creates a new Monitoring controller that is responsible for
// operating the monitoring components for all managed user clusters
func New(
	kubeClient kubernetes.Interface,
	userClusterConnProvider userClusterConnectionProvider,

	dc string,
	dcs map[string]provider.DatacenterMeta,
	overwriteRegistry string,
	nodePortRange string,
	nodeAccessNetwork string,
	etcdDiskSize string,
	inClusterPrometheusRulesFile string,
	inClusterPrometheusDisableDefaultRules bool,
	inClusterPrometheusDisableDefaultScrapingConfigs bool,
	inClusterPrometheusScrapingConfigsFile string,
	dockerPullConfigJSON []byte,

	clusterInformer kubermaticv1informers.ClusterInformer,
	serviceAccountInformer corev1informers.ServiceAccountInformer,
	configMapInformer corev1informers.ConfigMapInformer,
	roleInformer rbacv1informer.RoleInformer,
	roleBindingInformer rbacv1informer.RoleBindingInformer,
	serviceInformer corev1informers.ServiceInformer,
	statefulSetInformer appsv1informer.StatefulSetInformer,
	clusterRoleBindingInformer rbacv1informer.ClusterRoleBindingInformer,
	deploymentInformer appsv1informer.DeploymentInformer,
	secretInformer corev1informers.SecretInformer,
) (*Controller, error) {
	c := &Controller{
		kubeClient:              kubeClient,
		userClusterConnProvider: userClusterConnProvider,

		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, 5*time.Minute),
			"monitoring_cluster",
		),

		overwriteRegistry:                      overwriteRegistry,
		nodePortRange:                          nodePortRange,
		nodeAccessNetwork:                      nodeAccessNetwork,
		etcdDiskSize:                           resource.MustParse(etcdDiskSize),
		inClusterPrometheusRulesFile:           inClusterPrometheusRulesFile,
		inClusterPrometheusDisableDefaultRules: inClusterPrometheusDisableDefaultRules,
		inClusterPrometheusDisableDefaultScrapingConfigs: inClusterPrometheusDisableDefaultScrapingConfigs,
		inClusterPrometheusScrapingConfigsFile:           inClusterPrometheusScrapingConfigsFile,
		dockerPullConfigJSON:                             dockerPullConfigJSON,

		dc:  dc,
		dcs: dcs,
	}

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
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { c.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { c.handleChildObject(obj) },
	})
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { c.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { c.handleChildObject(obj) },
	})
	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { c.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { c.handleChildObject(obj) },
	})
	serviceAccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { c.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { c.handleChildObject(obj) },
	})
	roleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { c.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { c.handleChildObject(obj) },
	})
	roleBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { c.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { c.handleChildObject(obj) },
	})
	clusterRoleBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { c.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { c.handleChildObject(obj) },
	})

	c.clusterLister = clusterInformer.Lister()
	c.serviceLister = serviceInformer.Lister()
	c.configMapLister = configMapInformer.Lister()
	c.serviceAccountLister = serviceAccountInformer.Lister()
	c.deploymentLister = deploymentInformer.Lister()
	c.statefulSetLister = statefulSetInformer.Lister()
	c.roleLister = roleInformer.Lister()
	c.roleBindingLister = roleBindingInformer.Lister()
	c.clusterRoleBindingLister = clusterRoleBindingInformer.Lister()
	c.secretLister = secretInformer.Lister()

	return c, nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

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

	if err := c.sync(key.(string)); err != nil {
		glog.V(0).Infof("Error syncing %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return true
	}

	// Forget about the #AddRateLimited history of the key on every successful synchronization.
	// This ensures that future processing of updates for this key is not delayed because of
	// an outdated error history.
	c.queue.Forget(key)
	return true
}

func (c *Controller) enqueueAfter(cluster *kubermaticv1.Cluster, duration time.Duration) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(cluster)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", cluster, err))
		return
	}

	c.queue.AddAfter(key, duration)
}

func (c *Controller) enqueue(cluster *kubermaticv1.Cluster) {
	c.enqueueAfter(cluster, 0)
}

func (c *Controller) handleChildObject(obj interface{}) {
	object, ok := obj.(metav1.Object)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
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

	glog.V(4).Infof("syncing cluster %s", key)
	cluster := clusterFromCache.DeepCopy()

	if !clusterFromCache.Status.Health.AllHealthy() {
		glog.V(6).Infof("waiting for cluster %s to become healthy", key)
		c.enqueueAfter(cluster, healthCheckPeriod)
		return nil
	}

	if cluster.DeletionTimestamp != nil {
		// Cluster got deleted - all monitoring components will die soon
		return nil
	}

	data, err := c.getClusterTemplateData(cluster)
	if err != nil {
		return err
	}

	client, err := c.userClusterConnProvider.GetClient(cluster)
	if err != nil {
		return err
	}

	// check that all cluster roles in the user cluster are created
	if err = c.userClusterEnsureClusterRoles(cluster, data, client); err != nil {
		return err
	}

	// check that all cluster role bindings in the user cluster are created
	if err = c.userClusterEnsureClusterRoleBindings(cluster, data, client); err != nil {
		return err
	}

	// check that all service accounts are created
	if err := c.ensureServiceAccounts(cluster, data); err != nil {
		return err
	}

	// check that all roles are created
	if err := c.ensureRoles(cluster, data); err != nil {
		return err
	}

	// check that all role bindings are created
	if err := c.ensureRoleBindings(cluster, data); err != nil {
		return err
	}

	// check that all secrets are created
	if err := c.ensureSecrets(cluster, data); err != nil {
		return err
	}

	// check that all services are available
	if err := c.ensureServices(cluster, data); err != nil {
		return err
	}

	// check that all ConfigMaps are available
	if err := c.ensureConfigMaps(cluster, data); err != nil {
		return err
	}

	// check that all Deployments are available
	if err := c.ensureDeployments(cluster, data); err != nil {
		return err
	}

	// check that all StatefulSets are created
	return c.ensureStatefulSets(cluster, data)
}
