package cluster

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	k8cuserclusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/clusterdeletion"
	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
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
	GetClient(*kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Features struct {
	VPA                          bool
	EtcdDataCorruptionChecks     bool
	KubernetesOIDCAuthentication bool
}

// Reconciler is a controller which is responsible for managing clusters
type Reconciler struct {
	ctrlruntimeclient.Client
	log                     *zap.SugaredLogger
	userClusterConnProvider userClusterConnectionProvider
	workerName              string

	externalURL string
	seedGetter  provider.SeedGetter

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
	kubermaticImage                                  string
	dnatControllerImage                              string
	concurrentClusterUpdates                         int

	oidcCAFile         string
	oidcIssuerURL      string
	oidcIssuerClientID string

	features Features
}

// NewController creates a cluster controller.
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	externalURL string,
	seedGetter provider.SeedGetter,
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
	nodeLocalDNSCacheEnabled bool,
	concurrentClusterUpdates int,

	oidcCAFile string,
	oidcIssuerURL string,
	oidcIssuerClientID string,
	kubermaticImage string,
	dnatControllerImage string,
	features Features) error {

	reconciler := &Reconciler{
		log:                     log.Named(ControllerName),
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
		kubermaticImage:                                  kubermaticImage,
		dnatControllerImage:                              dnatControllerImage,
		concurrentClusterUpdates:                         concurrentClusterUpdates,

		externalURL: externalURL,
		seedGetter:  seedGetter,

		oidcCAFile:         oidcCAFile,
		oidcIssuerURL:      oidcIssuerURL,
		oidcIssuerClientID: oidcIssuerClientID,

		features: features,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

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
		if err := c.Watch(&source.Kind{Type: t}, controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kubeapierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	log = log.With("cluster", cluster.Name)

	if cluster.Spec.Pause {
		log.Debug("Skipping because the cluster is paused")
		return reconcile.Result{}, nil
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		log.Debugw(
			"Skipping because the cluster has a different worker name set",
			"cluster-worker-name", cluster.Labels[kubermaticv1.WorkerNameLabelKey],
		)
		return reconcile.Result{}, nil
	}

	if cluster.Annotations["kubermatic.io/openshift"] != "" {
		log.Debug("Skipping because the cluster is an OpenShift cluster")
		return reconcile.Result{}, nil
	}

	// only reconcile this cluster if there are not yet too many updates running
	if available, err := controllerutil.ClusterAvailableForReconciling(ctx, r, cluster, r.concurrentClusterUpdates); !available || err != nil {
		log.Infow("Concurrency limit reached, checking again in 10 seconds", "concurrency-limit", r.concurrentClusterUpdates)
		return reconcile.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}

	successfullyReconciled := true
	// Add a wrapping here so we can emit an event on error
	result, reconcileErr := r.reconcile(ctx, log, cluster)
	if reconcileErr != nil {
		successfullyReconciled = false
		log.Errorw("Reconciling failed", zap.Error(reconcileErr))
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", reconcileErr)
	}

	if result == nil {
		result = &reconcile.Result{}
	}

	hasUnfinishedUpdates, err := controllerutil.SetSeedResourcesUpToDateCondition(ctx, cluster, r.Client, successfullyReconciled)
	if err != nil {
		log.Errorw("failed to update clusters status conditions", zap.Error(err))
		reconcileErr = fmt.Errorf("failed to set cluster status: %v after reconciliation was done with err=%v", err, reconcileErr)
	}

	if !hasUnfinishedUpdates {
		cluster.Status.SetClusterCondition()
	}

	return *result, reconcileErr
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.DeletionTimestamp != nil {
		log.Debug("Cleaning up cluster")

		// Defer getting the client to make sure we only request it if we actually need it
		userClusterClientGetter := func() (ctrlruntimeclient.Client, error) {
			client, err := r.userClusterConnProvider.GetClient(cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to get user cluster client: %v", err)
			}
			return client, nil
		}
		// Always requeue a cluster after we executed the cleanup.
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, clusterdeletion.New(r.Client, userClusterClientGetter).CleanupCluster(ctx, log, cluster)
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
