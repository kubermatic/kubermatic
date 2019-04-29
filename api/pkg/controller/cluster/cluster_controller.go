package cluster

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	k8cuserclusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/clusterdeletion"
	kubermaticscheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	admissionregistrationclientset "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	aggregationclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_kubernetes_controller"
)

// userClusterConnectionProvider offers functions to retrieve clients for the given user clusters
type userClusterConnectionProvider interface {
	GetClient(*kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (kubernetes.Interface, error)
	GetApiextensionsClient(*kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (apiextensionsclientset.Interface, error)
	GetAdmissionRegistrationClient(*kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (admissionregistrationclientset.AdmissionregistrationV1beta1Interface, error)
	GetKubeAggregatorClient(*kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (aggregationclientset.Interface, error)
	GetDynamicClient(*kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Features struct {
	VPA                          bool
	EtcdDataCorruptionChecks     bool
	KubernetesOIDCAuthentication bool
}

// Reconciler is a controller which is responsible for managing clusters
type Reconciler struct {
	ctrlruntimeclient.Client
	userClusterConnProvider userClusterConnectionProvider
	workerName              string

	externalURL string
	dcs         map[string]provider.DatacenterMeta
	dc          string

	recorder record.EventRecorder

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
	nodeLocalDNSCacheEnabled                         bool

	oidcCAFile         string
	oidcIssuerURL      string
	oidcIssuerClientID string

	features Features
}

// NewController creates a cluster controller.
func Add(
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	externalURL string,
	dc string,
	dcs map[string]provider.DatacenterMeta,
	userClusterConnProvider userClusterConnectionProvider,
	overwriteRegistry string,
	nodePortRange string,
	nodeAccessNetwork string,
	etcdDiskSize resource.Quantity,
	monitoringScrapeAnnotationPrefix string,
	inClusterPrometheusRulesFile string,
	inClusterPrometheusDisableDefaultRules bool,
	inClusterPrometheusDisableDefaultScrapingConfigs bool,
	inClusterPrometheusScrapingConfigsFile string,
	dockerPullConfigJSON []byte,

	oidcCAFile string,
	oidcIssuerURL string,
	oidcIssuerClientID string,
	nodeLocalDNSCacheEnabled bool,
	features Features) error {

	if err := kubermaticscheme.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("failed to add kubermatic scheme: %v", err)
	}

	reconciler := &Reconciler{
		Client:                  mgr.GetClient(),
		userClusterConnProvider: userClusterConnProvider,
		workerName:              workerName,

		recorder: mgr.GetRecorder(ControllerName),

		overwriteRegistry:                      overwriteRegistry,
		nodePortRange:                          nodePortRange,
		nodeAccessNetwork:                      nodeAccessNetwork,
		etcdDiskSize:                           etcdDiskSize,
		inClusterPrometheusRulesFile:           inClusterPrometheusRulesFile,
		inClusterPrometheusDisableDefaultRules: inClusterPrometheusDisableDefaultRules,
		inClusterPrometheusDisableDefaultScrapingConfigs: inClusterPrometheusDisableDefaultScrapingConfigs,
		inClusterPrometheusScrapingConfigsFile:           inClusterPrometheusScrapingConfigsFile,
		monitoringScrapeAnnotationPrefix:                 monitoringScrapeAnnotationPrefix,
		dockerPullConfigJSON:                             dockerPullConfigJSON,
		nodeLocalDNSCacheEnabled:                         nodeLocalDNSCacheEnabled,

		externalURL: externalURL,
		dc:          dc,
		dcs:         dcs,

		oidcCAFile:         oidcCAFile,
		oidcIssuerURL:      oidcIssuerURL,
		oidcIssuerClientID: oidcIssuerClientID,

		features: features,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	ownerHandler := &handler.EnqueueRequestForOwner{IsController: true, OwnerType: &kubermaticv1.Cluster{}}

	typesToWatch := []runtime.Object{
		&corev1.Service{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Namespace{},
		&appsv1.StatefulSet{},
		&appsv1.Deployment{},
		&batchv1beta1.CronJob{},
		&policyv1beta1.PodDisruptionBudget{},
		&autoscalingv1beta2.VerticalPodAutoscaler{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, ownerHandler); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kubeapierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.Spec.Pause {
		glog.V(4).Infof("skipping paused cluster %s", cluster.Name)
		return reconcile.Result{}, nil
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return reconcile.Result{}, nil
	}

	if cluster.Annotations["kubermatic.io/openshift"] != "" {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := r.reconcile(ctx, cluster)
	if err != nil {
		glog.Errorf("Failed to reconcile cluster %s: %v", cluster.Name, err)
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	glog.V(4).Infof("Reconciling cluster %s", cluster.Name)

	if cluster.DeletionTimestamp != nil {
		if cluster.Status.Phase != kubermaticv1.DeletingClusterStatusPhase {
			err := r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
				c.Status.Phase = kubermaticv1.DeletingClusterStatusPhase
			})
			if err != nil {
				return nil, err
			}
		}

		userClusterClient, err := r.userClusterConnProvider.GetDynamicClient(cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to get user cluster client: %v", err)
		}
		return clusterdeletion.New(r.Client, userClusterClient).CleanupCluster(ctx, cluster)
	}

	if cluster.Status.Phase == kubermaticv1.NoneClusterStatusPhase {
		err := r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.Phase = kubermaticv1.ValidatingClusterStatusPhase
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Status.Phase == kubermaticv1.ValidatingClusterStatusPhase {
		err := r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.Phase = kubermaticv1.LaunchingClusterStatusPhase
		})
		if err != nil {
			return nil, err
		}
	}

	res, err := r.reconcileCluster(ctx, cluster)
	if err != nil {
		updateErr := r.updateClusterError(ctx, cluster, kubermaticv1.ReconcileClusterError, err.Error())
		if updateErr != nil {
			return nil, fmt.Errorf("failed to set the cluster error: %v", updateErr)
		}
		return nil, err
	}

	if err := r.clearClusterError(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to clear error on cluster: %v", err)
	}

	return res, nil
}

func (r *Reconciler) updateCluster(ctx context.Context, cluster *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster)) error {
	// Store it here because it may be unset later on if an update request failed
	name := cluster.Name
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		//Get latest version
		if err := r.Get(ctx, types.NamespacedName{Name: name}, cluster); err != nil {
			return err
		}
		// Apply modifications
		modify(cluster)
		// Update the cluster
		return r.Update(ctx, cluster)
	})
}

func (r *Reconciler) updateClusterError(ctx context.Context, cluster *kubermaticv1.Cluster, reason kubermaticv1.ClusterStatusError, message string) error {
	if cluster.Status.ErrorReason == nil || *cluster.Status.ErrorReason != reason {
		err := r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.ErrorMessage = &message
			c.Status.ErrorReason = &reason
		})
		if err != nil {
			return fmt.Errorf("failed to set error status on cluster to: errorReason=%q errorMessage=%q. Could not update cluster: %v", reason, message, err)
		}
	}

	return nil
}

func (r *Reconciler) clearClusterError(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if cluster.Status.ErrorReason != nil || cluster.Status.ErrorMessage != nil {
		err := r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.ErrorMessage = nil
			c.Status.ErrorReason = nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) getOwnerRefForCluster(cluster *kubermaticv1.Cluster) metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))
}
