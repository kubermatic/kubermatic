package kubeone

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/clusterdeletion"
	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1helper "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1/helper"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubeone_controller"
)

// Reconciler is a controller which is responsible for managing cluster objects managed by the KubeOne provider.
type Reconciler struct {
	client.Client
	log                     *zap.SugaredLogger
	userClusterConnProvider *clusterclient.Provider
	workerName              string

	seedGetter provider.SeedGetter

	recorder record.EventRecorder

	overwriteRegistry                                string
	nodePortRange                                    string
	nodeAccessNetwork                                string
	inClusterPrometheusRulesFile                     string
	inClusterPrometheusDisableDefaultRules           bool
	inClusterPrometheusDisableDefaultScrapingConfigs bool
	inClusterPrometheusScrapingConfigsFile           string
	monitoringScrapeAnnotationPrefix                 string
	dockerPullConfigJSON                             []byte
	nodeLocalDNSCacheEnabled                         bool
	kubermaticImage                                  string
	dnatControllerImage                              string
	etcdDiskSize                                     resource.Quantity
	concurrentClusterUpdates                         int

	oidcCAFile         string
	oidcIssuerURL      string
	oidcIssuerClientID string
}

func Add(mgr manager.Manager, log *zap.SugaredLogger, workerName string) error {
	r := &Reconciler{
		log:        log.Named(ControllerName),
		workerName: workerName,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
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
			log.Debug("Could not find cluster resource")
			return reconcile.Result{}, nil
		}
	}

	if !cluster.IsKubeOne() {
		log.Debug("Skipping because the cluster is not a Kubeone cluster")
		return reconcile.Result{}, nil
	}

	reconcileFunc := func() (*reconcile.Result, error) {
		if available, err := controllerutil.ClusterAvailableForReconciling(ctx, r, cluster, r.concurrentClusterUpdates); !available || err != nil {
			log.Infow("Concurrency limit reached, checking again in 10 seconds", "concurrency-limit", r.concurrentClusterUpdates)

			return &reconcile.Result{RequeueAfter: 10 * time.Second}, err
		}

		return r.reconcile(ctx, log, cluster)
	}

	// Add a wrapping here so we can emit an event on an error.
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		kubermaticv1.ClusterConditionClusterControllerReconcilingSuccess,
		reconcileFunc,
	)

	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	if result == nil {
		return reconcile.Result{}, err
	}

	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.DeletionTimestamp != nil {
		log.Debug("Cleaning up cluster")

		// Defer getting the client to make sure we only request it if we actually need it
		userClusterClientGetter := func() (client.Client, error) {
			client, err := r.userClusterConnProvider.GetClient(cluster)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get user cluster client")
			}

			log.Debugw("Getting client for cluster", "cluster", cluster.Name)
			return client, nil
		}

		// TODO: Implement our own Deletion struct (hint: convert the existing one to an interface and then implement a new one ;) )
		// THIS WONT WORK NOW.
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, clusterdeletion.New(r.Client, userClusterClientGetter).CleanupCluster(ctx, log, cluster)
	}

	if cluster.Spec.Cloud.Kubeone == nil {
		return nil, errors.New("kubeone cluster but .Spec.Cloud.KubeOne is nil")
	}

	if err := r.ensureNamespace(ctx, cluster); err != nil {
		return nil, errors.Wrap(err, "failed to reconcile namespace")
	}

	data, err := r.getClusterTemplateData(ctx, r.Client, cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster template data")
	}

	if err := r.ensureConfigMaps(ctx, cluster, data); err != nil {
		return nil, errors.Wrap(err, "failed to ensure configmaps")
	}

	if err := r.ensureDeployments(ctx, cluster, data); err != nil {
		return nil, errors.Wrap(err, "failed to ensure deployment")
	}

	return &reconcile.Result{}, nil
}

func (r *Reconciler) ensureNamespace(ctx context.Context, c *kubermaticv1.Cluster) error {
	name := c.Status.NamespaceName

	if name == "" {
		if err := r.updateCluster(ctx, c, func(c *kubermaticv1.Cluster) {
			name = fmt.Sprintf("cluster-%s", c.Name)
		}); err != nil {
			return errors.Wrap(err, "failed to set .Status.NamespaceName")
		}
	}

	// TODO: Update typesToWatch
	ns := &corev1.Namespace{}

	if err := r.Get(ctx, types.NamespacedName{Namespace: "", Name: name}, ns); err != nil {
		if !kubeapierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get Namespace %q", name)
		}

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(c, kubermaticv1.SchemeGroupVersion.WithKind("Cluster")),
				},
			},
		}

		if err := r.Create(ctx, ns); err != nil {
			return errors.Wrapf(err, "failed to create Namespace %q", name)
		}
	}

	return nil
}

func (r *Reconciler) updateCluster(ctx context.Context, c *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster)) error {
	oldCluster := c.DeepCopy()
	modify(c)
	return r.Patch(ctx, c, client.MergeFrom(oldCluster))
}
