package monitoring

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	k8cuserclusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// The monitoring controller waits for the cluster to become healthy,
	// before adding the monitoring components to the clusters
	healthCheckPeriod = 5 * time.Second

	ControllerName = "kubermatic_monitoring_controller"
)

// userClusterConnectionProvider offers functions to retrieve clients for the given user clusters
type userClusterConnectionProvider interface {
	GetClient(*kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

// Features describes the enabled features for the monitoring controller
type Features struct {
	VPA bool
}

// Reconciler stores all components required for monitoring
type Reconciler struct {
	ctrlruntimeclient.Client
	userClusterConnProvider userClusterConnectionProvider
	workerName              string

	log *zap.SugaredLogger

	recorder record.EventRecorder

	seedGetter                                       provider.SeedGetter
	overwriteRegistry                                string
	nodePortRange                                    string
	nodeAccessNetwork                                string
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
	dockerPullConfigJSON                             []byte
	// Annotation prefix to discover user cluster resources
	// example: kubermatic.io -> kubermatic.io/path,kubermatic.io/port
	monitoringScrapeAnnotationPrefix string
	nodeLocalDNSCacheEnabled         bool
	concurrentClusterUpdates         int

	features Features
}

// Add creates a new Monitoring controller that is responsible for
// operating the monitoring components for all managed user clusters
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,

	userClusterConnProvider userClusterConnectionProvider,
	seedGetter provider.SeedGetter,
	overwriteRegistry string,
	nodePortRange string,
	nodeAccessNetwork string,
	monitoringScrapeAnnotationPrefix string,
	inClusterPrometheusRulesFile string,
	inClusterPrometheusDisableDefaultRules bool,
	inClusterPrometheusDisableDefaultScrapingConfigs bool,
	inClusterPrometheusScrapingConfigsFile string,
	dockerPullConfigJSON []byte,
	nodeLocalDNSCacheEnabled bool,
	concurrentClusterUpdates int,

	features Features,
) error {
	log = log.Named(ControllerName)

	reconciler := &Reconciler{
		Client:                  mgr.GetClient(),
		userClusterConnProvider: userClusterConnProvider,
		workerName:              workerName,

		log: log,

		recorder: mgr.GetRecorder(ControllerName),

		overwriteRegistry:                                overwriteRegistry,
		nodePortRange:                                    nodePortRange,
		nodeAccessNetwork:                                nodeAccessNetwork,
		monitoringScrapeAnnotationPrefix:                 monitoringScrapeAnnotationPrefix,
		inClusterPrometheusRulesFile:                     inClusterPrometheusRulesFile,
		inClusterPrometheusDisableDefaultRules:           inClusterPrometheusDisableDefaultRules,
		inClusterPrometheusDisableDefaultScrapingConfigs: inClusterPrometheusDisableDefaultScrapingConfigs,
		inClusterPrometheusScrapingConfigsFile:           inClusterPrometheusScrapingConfigsFile,
		dockerPullConfigJSON:                             dockerPullConfigJSON,
		nodeLocalDNSCacheEnabled:                         nodeLocalDNSCacheEnabled,
		concurrentClusterUpdates:                         concurrentClusterUpdates,
		seedGetter:                                       seedGetter,

		features: features,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	typesToWatch := []runtime.Object{
		&corev1.ServiceAccount{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
		&corev1.Secret{},
		&corev1.ConfigMap{},
		&appsv1.Deployment{},
		&appsv1.StatefulSet{},
		&autoscalingv1beta2.VerticalPodAutoscaler{},
		&corev1.Service{},
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
			log.Errorw("Couldn't find cluster", zap.Error(err))
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	log = log.With("cluster", cluster.Name)

	if cluster.Spec.Pause {
		log.Debug("Skipping cluster reconciling because it was set to paused")
		return reconcile.Result{}, nil
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return reconcile.Result{}, nil
	}

	if cluster.DeletionTimestamp != nil {
		// Cluster got deleted - all monitoring components will be automatically garbage collected (Due to the ownerRef)
		return reconcile.Result{}, nil
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Skipping cluster reconciling because it has no namespace yet")
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	}

	// Wait until the UCCM is ready - otherwise we deploy with missing RBAC resources
	if kubermaticv1.HealthStatusDown == cluster.Status.ExtendedHealth.UserClusterControllerManager {
		log.Debug("Skipping cluster reconciling because the UserClusterControllerManager is not ready yet")
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
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
		log.Errorw("Failed to reconcile cluster", zap.Error(reconcileErr))
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", reconcileErr)
	}

	if result == nil {
		result = &reconcile.Result{}
	}

	if err := controllerutil.SetSeedResourcesUpToDateCondition(ctx, cluster, r, successfullyReconciled); err != nil {
		log.Errorw("failed to update clusters status conditions", zap.Error(err))
		reconcileErr = fmt.Errorf("failed to set cluster status: %v after reconciliation was done with err=%v", err, reconcileErr)
	}

	return *result, reconcileErr
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	log.Debug("Reconciling cluster now")

	data, err := r.getClusterTemplateData(context.Background(), r.Client, cluster)
	if err != nil {
		return nil, err
	}

	// check that all service accounts are created
	if err := r.ensureServiceAccounts(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all roles are created
	if err := r.ensureRoles(ctx, cluster); err != nil {
		return nil, err
	}

	// check that all role bindings are created
	if err := r.ensureRoleBindings(ctx, cluster); err != nil {
		return nil, err
	}

	// check that all secrets are created
	if err := r.ensureSecrets(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all ConfigMaps are available
	if err := r.ensureConfigMaps(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all Deployments are available
	if err := r.ensureDeployments(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all StatefulSets are created
	if err := r.ensureStatefulSets(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all VerticalPodAutoscaler's are created
	if err := r.ensureVerticalPodAutoscalers(ctx, cluster); err != nil {
		return nil, err
	}

	// check that all Services's are created
	if err := r.ensureServices(ctx, cluster, data); err != nil {
		return nil, err
	}

	log.Debug("Reconciliation completed successfully")

	return &reconcile.Result{}, nil
}
