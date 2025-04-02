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

package monitoring

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// The monitoring controller waits for the cluster to become healthy,
	// before adding the monitoring components to the clusters.
	healthCheckPeriod = 5 * time.Second

	ControllerName = "kkp-monitoring-controller"
)

// userClusterConnectionProvider offers functions to retrieve clients for the given user clusters.
type userClusterConnectionProvider interface {
	GetClient(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

// Features describes the enabled features for the monitoring controller.
type Features struct {
	VPA bool
}

// Reconciler stores all components required for monitoring.
type Reconciler struct {
	ctrlruntimeclient.Client

	userClusterConnProvider userClusterConnectionProvider
	workerName              string
	log                     *zap.SugaredLogger
	recorder                record.EventRecorder

	seedGetter               provider.SeedGetter
	configGetter             provider.KubermaticConfigurationGetter
	overwriteRegistry        string
	nodeAccessNetwork        string
	dockerPullConfigJSON     []byte
	concurrentClusterUpdates int

	features Features
	versions kubermatic.Versions
}

// Add creates a new Monitoring controller that is responsible for
// operating the monitoring components for all managed user clusters.
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,

	userClusterConnProvider userClusterConnectionProvider,
	seedGetter provider.SeedGetter,
	configGetter provider.KubermaticConfigurationGetter,
	overwriteRegistry string,
	nodeAccessNetwork string,
	dockerPullConfigJSON []byte,
	concurrentClusterUpdates int,

	features Features,
	versions kubermatic.Versions,
) error {
	log = log.Named(ControllerName)

	reconciler := &Reconciler{
		Client:                  mgr.GetClient(),
		userClusterConnProvider: userClusterConnProvider,
		workerName:              workerName,
		log:                     log,
		recorder:                mgr.GetEventRecorderFor(ControllerName),

		overwriteRegistry:        overwriteRegistry,
		nodeAccessNetwork:        nodeAccessNetwork,
		dockerPullConfigJSON:     dockerPullConfigJSON,
		concurrentClusterUpdates: concurrentClusterUpdates,
		seedGetter:               seedGetter,
		configGetter:             configGetter,

		features: features,
		versions: versions,
	}

	bldr := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{})

	for _, t := range []ctrlruntimeclient.Object{
		&corev1.ServiceAccount{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
		&corev1.Secret{},
		&corev1.ConfigMap{},
		&appsv1.Deployment{},
		&appsv1.StatefulSet{},
		&autoscalingv1.VerticalPodAutoscaler{},
		&corev1.Service{},
	} {
		bldr.Watches(t, controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient()))
	}

	_, err := bldr.Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	log = log.With("cluster", cluster.Name)

	if cluster.DeletionTimestamp != nil {
		// Cluster got deleted - all monitoring components will be automatically garbage collected (Due to the ownerRef)
		return reconcile.Result{}, nil
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Skipping cluster reconciling because it has no namespace yet")
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	}

	// Wait until the UCCM is ready - otherwise we deploy with missing RBAC resources
	if cluster.Status.ExtendedHealth.UserClusterControllerManager != kubermaticv1.HealthStatusUp {
		log.Debug("Skipping cluster reconciling because the UserClusterControllerManager is not ready yet")
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := controllerutil.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMonitoringControllerReconcilingSuccess,
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

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	log.Debug("Reconciling cluster now")

	data, err := r.getClusterTemplateData(ctx, r, cluster)
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
