package cluster

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsv1informer "k8s.io/client-go/informers/apps/v1"
	batchv1beta1informer "k8s.io/client-go/informers/batch/v1beta1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	extensionsv1beta1informers "k8s.io/client-go/informers/extensions/v1beta1"
	policyv1beta1informers "k8s.io/client-go/informers/policy/v1beta1"
	rbacv1informer "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	admissionregistrationclientset "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	appsv1lister "k8s.io/client-go/listers/apps/v1"
	batchv1beta1lister "k8s.io/client-go/listers/batch/v1beta1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	extensionsv1beta1lister "k8s.io/client-go/listers/extensions/v1beta1"
	policyv1beta1lister "k8s.io/client-go/listers/policy/v1beta1"
	rbacb1lister "k8s.io/client-go/listers/rbac/v1"
	aggregationclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"

	clusterv1alpha1clientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

// userClusterConnectionProvider offers functions to retrieve clients for the given user clusters
type userClusterConnectionProvider interface {
	GetClient(*kubermaticv1.Cluster) (kubernetes.Interface, error)
	GetMachineClient(*kubermaticv1.Cluster) (clusterv1alpha1clientset.Interface, error)
	GetApiextensionsClient(*kubermaticv1.Cluster) (apiextensionsclientset.Interface, error)
	GetAdmissionRegistrationClient(*kubermaticv1.Cluster) (admissionregistrationclientset.AdmissionregistrationV1beta1Interface, error)
	GetKubeAggregatorClient(*kubermaticv1.Cluster) (aggregationclientset.Interface, error)
}

// Controller is a controller which is responsible for managing clusters
type Controller struct {
	kubermaticClient        kubermaticclientset.Interface
	kubeClient              kubernetes.Interface
	userClusterConnProvider userClusterConnectionProvider

	externalURL string
	dcs         map[string]provider.DatacenterMeta
	dc          string
	cps         map[string]provider.CloudProvider

	queue workqueue.RateLimitingInterface

	overwriteRegistry                                string
	nodePortRange                                    string
	nodeAccessNetwork                                string
	etcdDiskSize                                     resource.Quantity
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
	monitoringScrapeAnnotationPrefix                 string
	dockerPullConfigJSON                             []byte

	clusterLister             kubermaticv1lister.ClusterLister
	namespaceLister           corev1lister.NamespaceLister
	secretLister              corev1lister.SecretLister
	serviceLister             corev1lister.ServiceLister
	pvcLister                 corev1lister.PersistentVolumeClaimLister
	configMapLister           corev1lister.ConfigMapLister
	serviceAccountLister      corev1lister.ServiceAccountLister
	deploymentLister          appsv1lister.DeploymentLister
	statefulSetLister         appsv1lister.StatefulSetLister
	cronJobLister             batchv1beta1lister.CronJobLister
	ingressLister             extensionsv1beta1lister.IngressLister
	roleLister                rbacb1lister.RoleLister
	roleBindingLister         rbacb1lister.RoleBindingLister
	clusterRoleBindingLister  rbacb1lister.ClusterRoleBindingLister
	podDisruptionBudgetLister policyv1beta1lister.PodDisruptionBudgetLister

	skipOidcTLSVerify   bool
	oidcIssuerURL       string
	oidcIssuerClientID  string
	oidcDexSecretCAName string
}

// NewController creates a cluster controller.
func NewController(
	kubeClient kubernetes.Interface,
	kubermaticClient kubermaticclientset.Interface,
	externalURL string,
	dc string,
	dcs map[string]provider.DatacenterMeta,
	cps map[string]provider.CloudProvider,
	userClusterConnProvider userClusterConnectionProvider,
	overwriteRegistry string,
	nodePortRange string,
	nodeAccessNetwork string,
	etcdDiskSize string,
	monitoringScrapeAnnotationPrefix string,
	inClusterPrometheusRulesFile string,
	inClusterPrometheusDisableDefaultRules bool,
	inClusterPrometheusDisableDefaultScrapingConfigs bool,
	inClusterPrometheusScrapingConfigsFile string,
	dockerPullConfigJSON []byte,

	clusterInformer kubermaticv1informers.ClusterInformer,
	namespaceInformer corev1informers.NamespaceInformer,
	secretInformer corev1informers.SecretInformer,
	serviceInformer corev1informers.ServiceInformer,
	pvcInformer corev1informers.PersistentVolumeClaimInformer,
	configMapInformer corev1informers.ConfigMapInformer,
	serviceAccountInformer corev1informers.ServiceAccountInformer,
	deploymentInformer appsv1informer.DeploymentInformer,
	statefulSetInformer appsv1informer.StatefulSetInformer,
	cronJobInformer batchv1beta1informer.CronJobInformer,
	ingressInformer extensionsv1beta1informers.IngressInformer,
	roleInformer rbacv1informer.RoleInformer,
	roleBindingInformer rbacv1informer.RoleBindingInformer,
	clusterRoleBindingInformer rbacv1informer.ClusterRoleBindingInformer,
	podDisruptionBudgetInformer policyv1beta1informers.PodDisruptionBudgetInformer,
	skipOidcTLSVerify bool,
	oidcIssuerURL string,
	oidcIssuerClientID string,
	oidcDexSecretCAName string) (*Controller, error) {
	cc := &Controller{
		kubermaticClient:        kubermaticClient,
		kubeClient:              kubeClient,
		userClusterConnProvider: userClusterConnProvider,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, 5*time.Minute), "cluster"),

		overwriteRegistry:                      overwriteRegistry,
		nodePortRange:                          nodePortRange,
		nodeAccessNetwork:                      nodeAccessNetwork,
		etcdDiskSize:                           resource.MustParse(etcdDiskSize),
		inClusterPrometheusRulesFile:           inClusterPrometheusRulesFile,
		inClusterPrometheusDisableDefaultRules: inClusterPrometheusDisableDefaultRules,
		inClusterPrometheusDisableDefaultScrapingConfigs: inClusterPrometheusDisableDefaultScrapingConfigs,
		inClusterPrometheusScrapingConfigsFile:           inClusterPrometheusScrapingConfigsFile,
		monitoringScrapeAnnotationPrefix:                 monitoringScrapeAnnotationPrefix,
		dockerPullConfigJSON:                             dockerPullConfigJSON,

		externalURL: externalURL,
		dc:          dc,
		dcs:         dcs,
		cps:         cps,

		skipOidcTLSVerify:   skipOidcTLSVerify,
		oidcIssuerURL:       oidcIssuerURL,
		oidcIssuerClientID:  oidcIssuerClientID,
		oidcDexSecretCAName: oidcDexSecretCAName,
	}

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
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
					runtime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				cluster, ok = tombstone.Obj.(*kubermaticv1.Cluster)
				if !ok {
					runtime.HandleError(fmt.Errorf("tombstone contained object that is not a Cluster %#v", obj))
					return
				}
			}
			cc.enqueue(cluster)
		},
	})

	//In case one of our child objects change, we should update our state
	namespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	pvcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	serviceAccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	roleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	roleBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	clusterRoleBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	podDisruptionBudgetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})
	cronJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { cc.handleChildObject(obj) },
		UpdateFunc: func(old, cur interface{}) { cc.handleChildObject(cur) },
		DeleteFunc: func(obj interface{}) { cc.handleChildObject(obj) },
	})

	cc.clusterLister = clusterInformer.Lister()
	cc.namespaceLister = namespaceInformer.Lister()
	cc.secretLister = secretInformer.Lister()
	cc.serviceLister = serviceInformer.Lister()
	cc.pvcLister = pvcInformer.Lister()
	cc.configMapLister = configMapInformer.Lister()
	cc.serviceAccountLister = serviceAccountInformer.Lister()
	cc.deploymentLister = deploymentInformer.Lister()
	cc.statefulSetLister = statefulSetInformer.Lister()
	cc.cronJobLister = cronJobInformer.Lister()
	cc.ingressLister = ingressInformer.Lister()
	cc.roleLister = roleInformer.Lister()
	cc.roleBindingLister = roleBindingInformer.Lister()
	cc.clusterRoleBindingLister = clusterRoleBindingInformer.Lister()
	cc.podDisruptionBudgetLister = podDisruptionBudgetInformer.Lister()

	return cc, nil
}

func (cc *Controller) enqueueAfter(cluster *kubermaticv1.Cluster, duration time.Duration) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(cluster)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", cluster, err))
		return
	}

	cc.queue.AddAfter(key, duration)
}

func (cc *Controller) enqueue(cluster *kubermaticv1.Cluster) {
	cc.enqueueAfter(cluster, 0)
}

func (cc *Controller) updateCluster(name string, modify func(*kubermaticv1.Cluster)) (updatedCluster *kubermaticv1.Cluster, err error) {
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		//Get latest version from cache
		cacheCluster, err := cc.clusterLister.Get(name)
		if err != nil {
			return err
		}

		currentCluster := cacheCluster.DeepCopy()
		// Apply modifications
		modify(currentCluster)
		// Update the cluster
		updatedCluster, err = cc.kubermaticClient.KubermaticV1().Clusters().Update(currentCluster)
		if err != nil && kubeapierrors.IsConflict(err) {
			//Get latest version from api
			currentCluster, err := cc.kubermaticClient.KubermaticV1().Clusters().Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// Apply modifications
			modify(currentCluster)
			// Update the cluster
			updatedCluster, err = cc.kubermaticClient.KubermaticV1().Clusters().Update(currentCluster)
			if err != nil {
				return err
			}
		}

		return err
	})

	return updatedCluster, err
}

func (cc *Controller) updateClusterError(cluster *kubermaticv1.Cluster, reason kubermaticv1.ClusterStatusError, message string) (*kubermaticv1.Cluster, error) {
	var err error
	if cluster.Status.ErrorReason == nil || *cluster.Status.ErrorReason != reason {
		cluster, err = cc.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
			c.Status.ErrorMessage = &message
			c.Status.ErrorReason = &reason
		})
		if err != nil {
			return nil, fmt.Errorf("failed to set error status on cluster to: errorReason='%s' errorMessage='%s'. Could not update cluster: %v", reason, message, err)
		}
	}

	return cluster, nil
}

func (cc *Controller) clearClusterError(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var err error
	if cluster.Status.ErrorReason != nil || cluster.Status.ErrorMessage != nil {
		cluster, err = cc.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
			c.Status.ErrorMessage = nil
			c.Status.ErrorReason = nil
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func (cc *Controller) syncCluster(key string) error {
	listerCluster, err := cc.clusterLister.Get(key)
	if err != nil {
		if kubeapierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("unable to retrieve cluster %q: %v", key, err)
	}

	cluster := listerCluster.DeepCopy()

	if cluster.Spec.Pause {
		glog.V(6).Infof("skipping paused cluster %s", key)
		return nil
	}

	glog.V(4).Infof("syncing cluster %s", key)

	if cluster.DeletionTimestamp != nil {
		if cluster.Status.Phase != kubermaticv1.DeletingClusterStatusPhase {
			cluster, err = cc.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
				c.Status.Phase = kubermaticv1.DeletingClusterStatusPhase
			})
			if err != nil {
				return err
			}
		}

		if cluster, err = cc.cleanupCluster(cluster); err != nil {
			return err
		}
		//Always requeue until the cluster is deleted
		cc.enqueueAfter(cluster, 10*time.Second)
		return nil
	}

	if cluster.Status.Phase == kubermaticv1.NoneClusterStatusPhase {
		cluster, err = cc.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
			c.Status.Phase = kubermaticv1.ValidatingClusterStatusPhase
		})
		if err != nil {
			return err
		}
	}

	if cluster.Status.Phase == kubermaticv1.ValidatingClusterStatusPhase {
		cluster, err = cc.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
			c.Status.Phase = kubermaticv1.LaunchingClusterStatusPhase
		})
		if err != nil {
			return err
		}
	}

	if _, err = cc.reconcileCluster(cluster); err != nil {
		_, updateErr := cc.updateClusterError(cluster, kubermaticv1.ReconcileClusterError, err.Error())
		if updateErr != nil {
			return fmt.Errorf("failed to set the cluster error: %v", updateErr)
		}
		return err
	}

	if _, err = cc.clearClusterError(cluster); err != nil {
		return fmt.Errorf("failed to clear error on cluster: %v", err)
	}

	return nil
}

func (cc *Controller) runWorker() {
	for cc.processNextItem() {
	}
}

func (cc *Controller) processNextItem() bool {
	key, quit := cc.queue.Get()
	if quit {
		return false
	}
	defer cc.queue.Done(key)

	if err := cc.syncCluster(key.(string)); err != nil {
		glog.V(0).Infof("Error syncing %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		cc.queue.AddRateLimited(key)
		return true
	}

	// Forget about the #AddRateLimited history of the key on every successful synchronization.
	// This ensures that future processing of updates for this key is not delayed because of
	// an outdated error history.
	cc.queue.Forget(key)
	return true
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (cc *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	for i := 0; i < workerCount; i++ {
		go wait.Until(cc.runWorker, time.Second, stopCh)
	}
	workers.Set(float64(workerCount))

	<-stopCh
}

func (cc *Controller) handleChildObject(i interface{}) {
	obj, ok := i.(metav1.Object)
	//Object might be a tombstone
	if !ok {
		tombstone, ok := i.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("couldn't get obj from tombstone %#v", obj))
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
		c, err := cc.clusterLister.Get(controllerRef.Name)
		if err != nil {
			if kubeapierrors.IsNotFound(err) {
				// No need to log something when the object gets deleted - happens when the gc cleans up after cluster deletion
				if obj.GetDeletionTimestamp() != nil {
					return
				}
				// If the child exists but the parent is gone, the GarbageCollector has not deleted this resource yet.
				glog.V(6).Infof("orphaned child obj found '%s/%s'. Responsible controller %s not found. GarbageCollector will take care.", obj.GetNamespace(), obj.GetName(), controllerRef.Name)
				return
			}
			runtime.HandleError(fmt.Errorf("failed to get cluster %s from lister: %v", controllerRef.Name, err))
			return
		}

		cc.enqueue(c)
		return
	}
}

func (cc *Controller) getOwnerRefForCluster(c *kubermaticv1.Cluster) metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(c, gv.WithKind("Cluster"))
}
