/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/clusterdeletion"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-kubernetes-controller"
)

// userClusterConnectionProvider offers functions to retrieve clients for the given user clusters.
type userClusterConnectionProvider interface {
	GetClient(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Features struct {
	VPA                          bool
	EtcdDataCorruptionChecks     bool
	KubernetesOIDCAuthentication bool
	EtcdLauncher                 bool
	DynamicResourceAllocation    bool
}

// Reconciler is a controller which is responsible for managing clusters.
type Reconciler struct {
	ctrlruntimeclient.Client

	log                     *zap.SugaredLogger
	userClusterConnProvider userClusterConnectionProvider
	workerName              string

	externalURL  string
	seedGetter   provider.SeedGetter
	configGetter provider.KubermaticConfigurationGetter

	recorder events.EventRecorder

	overwriteRegistry                string
	nodeAccessNetwork                string
	etcdDiskSize                     resource.Quantity
	userClusterMLAEnabled            bool
	dockerPullConfigJSON             []byte
	kubermaticImage                  string
	etcdLauncherImage                string
	dnatControllerImage              string
	machineControllerImageTag        string
	machineControllerImageRepository string
	concurrentClusterUpdates         int
	backupSchedule                   time.Duration
	backupCount                      int

	oidcIssuerURL      string
	oidcIssuerClientID string

	features Features
	versions kubermatic.Versions

	tunnelingAgentIP string
	caBundle         *certificates.CABundle
}

// Add creates a cluster controller.
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	externalURL string,
	seedGetter provider.SeedGetter,
	configGetter provider.KubermaticConfigurationGetter,
	userClusterConnProvider userClusterConnectionProvider,
	overwriteRegistry string,
	nodeAccessNetwork string,
	etcdDiskSize resource.Quantity,
	userClusterMLAEnabled bool,
	dockerPullConfigJSON []byte,
	concurrentClusterUpdates int,
	backupSchedule time.Duration,
	backupCount int,

	oidcIssuerURL string,
	oidcIssuerClientID string,
	kubermaticImage string,
	etcdLauncherImage string,
	machineControllerImageTag string,
	machineControllerImageRepository string,
	tunnelingAgentIP string,
	caBundle *certificates.CABundle,

	features Features,
	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		log:                     log.Named(ControllerName),
		userClusterConnProvider: userClusterConnProvider,
		workerName:              workerName,

		recorder: mgr.GetEventRecorder(ControllerName),

		overwriteRegistry:                overwriteRegistry,
		nodeAccessNetwork:                nodeAccessNetwork,
		etcdDiskSize:                     etcdDiskSize,
		userClusterMLAEnabled:            userClusterMLAEnabled,
		dockerPullConfigJSON:             dockerPullConfigJSON,
		kubermaticImage:                  kubermaticImage,
		etcdLauncherImage:                etcdLauncherImage,
		dnatControllerImage:              dnatControllerImage,
		machineControllerImageTag:        machineControllerImageTag,
		machineControllerImageRepository: machineControllerImageRepository,
		concurrentClusterUpdates:         concurrentClusterUpdates,
		backupSchedule:                   backupSchedule,
		backupCount:                      backupCount,

		externalURL:  externalURL,
		seedGetter:   seedGetter,
		configGetter: configGetter,

		oidcIssuerURL:      oidcIssuerURL,
		oidcIssuerClientID: oidcIssuerClientID,

		tunnelingAgentIP: tunnelingAgentIP,
		caBundle:         caBundle,

		features: features,
		versions: versions,
	}

	typesToWatch := []ctrlruntimeclient.Object{
		&corev1.Service{},
		&corev1.ServiceAccount{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Namespace{},
		&appsv1.StatefulSet{},
		&appsv1.Deployment{},
		&batchv1.CronJob{},
		&policyv1.PodDisruptionBudget{},
		&autoscalingv1.VerticalPodAutoscaler{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
		&networkingv1.NetworkPolicy{},
	}

	// During cluster deletions, we do not care about changes that happen inside the cluster namespace.
	// We would not be reconciling anything and we also do not want to re-trigger the cleanup every time
	// a Secret or Pod is deleted (instead we want to wait 10 seconds between deletion checks).
	// Instead of splitting this controller into 2 reconcilers, we simply do not return any requests if
	// the cluster is in deletion.
	inNamespaceHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		cluster, err := kubernetes.ClusterFromNamespace(ctx, reconciler, a.GetNamespace())
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %w", err))
			return []reconcile.Request{}
		}

		// if the cluster is already being deleted,
		// we do not care about the resources inside its namespace
		if cluster != nil && cluster.DeletionTimestamp == nil {
			return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
		}

		return []reconcile.Request{}
	})

	bldr := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{})

	for _, t := range typesToWatch {
		bldr.Watches(t, inNamespaceHandler)
	}

	_, err := bldr.Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	// do not use the request itself, as it might contain the namespace marker
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// the update controller needs to determine the target version based on the spec
	// before we can reconcile anything
	if cluster.Status.Versions.ControlPlane == "" {
		log.Debug("Cluster not yet ready for reconciling")

		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := controllerutil.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionClusterControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			// only reconcile this cluster if there are not yet too many updates running
			available, err := controllerutil.ClusterAvailableForReconciling(ctx, r, cluster, r.concurrentClusterUpdates)
			if err != nil {
				return &reconcile.Result{}, err
			}

			if !available {
				log.Infow("Concurrency limit reached, checking again in 10 seconds", "concurrency-limit", r.concurrentClusterUpdates)
				return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
			}

			return r.reconcile(ctx, log, cluster)
		},
	)

	// in case of errors, always return a zero result
	if result == nil {
		result = &reconcile.Result{}
	}

	// no need to log the error, controller-runtime does it for us
	if err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.DeletionTimestamp != nil {
		log.Debug("Cleaning up cluster")

		// Defer getting the client to make sure we only request it if we actually need it
		userClusterClientGetter := func() (ctrlruntimeclient.Client, error) {
			client, err := r.userClusterConnProvider.GetClient(ctx, cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to get user cluster client: %w", err)
			}
			return client, nil
		}

		if err := clusterdeletion.New(r, r.recorder, userClusterClientGetter).CleanupCluster(ctx, log, cluster); err != nil {
			return nil, err
		}

		// Always requeue a cluster after we executed the cleanup.
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	namespace, err := r.reconcileClusterNamespace(ctx, log, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure cluster namespace: %w", err)
	}

	// synchronize cluster.status.health for Kubernetes clusters
	if err := r.syncHealth(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to sync health: %w", err)
	}

	res, err := r.reconcileCluster(ctx, cluster, namespace)
	if err != nil {
		updateErr := r.updateClusterError(ctx, cluster, kubermaticv1.ReconcileClusterError, err.Error())
		if updateErr != nil {
			return nil, fmt.Errorf("failed to set the cluster error: %w", updateErr)
		}
		return nil, fmt.Errorf("failed to reconcile cluster: %w", err)
	}

	if err := r.clearClusterError(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to clear error on cluster: %w", err)
	}

	return res, nil
}

func (r *Reconciler) updateCluster(ctx context.Context, cluster *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster), opts ...ctrlruntimeclient.MergeFromOption) error {
	oldCluster := cluster.DeepCopy()
	modify(cluster)
	if reflect.DeepEqual(oldCluster, cluster) {
		return nil
	}

	if !reflect.DeepEqual(oldCluster.Status, cluster.Status) {
		return errors.New("updateCluster must not change cluster status")
	}

	if err := r.Patch(ctx, cluster, ctrlruntimeclient.MergeFromWithOptions(oldCluster, opts...)); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) AddFinalizers(ctx context.Context, cluster *kubermaticv1.Cluster, finalizers ...string) (*reconcile.Result, error) {
	return &reconcile.Result{}, kuberneteshelper.TryAddFinalizer(ctx, r, cluster, finalizers...)
}

func (r *Reconciler) updateClusterError(ctx context.Context, cluster *kubermaticv1.Cluster, reason kubermaticv1.ClusterStatusError, message string) error {
	err := controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ErrorMessage = &message
		c.Status.ErrorReason = &reason
	})
	if err != nil {
		return fmt.Errorf("failed to set error status on cluster to: errorReason=%q errorMessage=%q. Could not update cluster: %w", reason, message, err)
	}

	return nil
}

func (r *Reconciler) clearClusterError(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	return controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ErrorMessage = nil
		c.Status.ErrorReason = nil
	})
}

func (r *Reconciler) getOwnerRefForCluster(cluster *kubermaticv1.Cluster) metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))
}
