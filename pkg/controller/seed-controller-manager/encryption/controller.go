/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package encryption

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_encryption_controller"
)

// userClusterConnectionProvider offers functions to retrieve clients for the given user clusters
type userClusterConnectionProvider interface {
	GetClient(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Reconciler struct {
	ctrlruntimeclient.Client
	log                     *zap.SugaredLogger
	userClusterConnProvider userClusterConnectionProvider
	workerName              string
	recorder                record.EventRecorder

	versions kubermatic.Versions
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	userClusterConnProvider userClusterConnectionProvider,
	versions kubermatic.Versions) error {

	reconciler := &Reconciler{
		log:                     log.Named(ControllerName),
		Client:                  mgr.GetClient(),
		userClusterConnProvider: userClusterConnProvider,
		workerName:              workerName,

		recorder: mgr.GetEventRecorderFor(ControllerName),

		versions: versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &corev1.Secret{}},
		controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient()),
		predicateutil.ByName(resources.EncryptionConfigurationSecretName),
	); err != nil {
		return fmt.Errorf("failed to create watcher for corev1.Secret: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &appsv1.Deployment{}},
		controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient()),
		predicateutil.ByName(resources.ApiserverDeploymentName),
	); err != nil {
		return fmt.Errorf("failed to create watcher for appsv1.Deployment: %w", err)
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster := o.(*kubermaticv1.Cluster)
		return cluster.Spec.Features[kubermaticv1.ClusterFeatureEncryptionAtRest]
	}))
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	log = log.With("cluster", cluster.Name)

	// return early if encryption is not enabled in spec and EncryptionIntialized condition is not set
	// TODO: make a predicate?
	if cluster.Spec.EncryptionConfiguration == nil || !cluster.Spec.EncryptionConfiguration.Enabled || cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionFalse) {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionClusterControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	if result == nil {
		result = &reconcile.Result{}
	}

	return *result, err

}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// reconcile until encryption is successfully initialized
	if cluster.Spec.EncryptionConfiguration.Enabled && !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionTrue) {
		return r.setInitializedCondition(ctx, cluster)
	}

	// wait for apiserver rollout to complete before updating status
	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	primaryKey := fmt.Sprintf("secretbox:%s", cluster.Spec.EncryptionConfiguration.Secretbox.Keys[0].Name)
	if cluster.Status.ActiveEncryptionKey != primaryKey {
		if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.ActiveEncryptionKey = primaryKey
			kubermaticv1helper.SetClusterCondition(
				cluster,
				r.versions,
				kubermaticv1.ClusterConditionEncryptionFinished,
				corev1.ConditionFalse,
				"",
				fmt.Sprintf("Data re-encryption with key '%s' is pending", primaryKey),
			)
		}); err != nil {
			return &reconcile.Result{}, err
		}
	}

	// TODO: run re-encryption
	if cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionFinished, corev1.ConditionFalse) {
		return r.encryptData(ctx, cluster)
	}

	return &reconcile.Result{}, nil
}

func (r *Reconciler) setInitializedCondition(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		return &reconcile.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	var deployment appsv1.Deployment
	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      resources.ApiserverDeploymentName,
		Namespace: cluster.Status.NamespaceName,
	}, &deployment); err != nil {
		return &reconcile.Result{}, err
	}

	hasSecretVolume := false
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Secret != nil && volume.Secret.SecretName == resources.EncryptionConfigurationSecretName {
			hasSecretVolume = true
		}
	}

	if !hasSecretVolume {
		return &reconcile.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	// set the encryption initialized condition (should only happen once on every cluster)
	if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
		kubermaticv1helper.SetClusterCondition(
			cluster,
			r.versions,
			kubermaticv1.ClusterConditionEncryptionInitialized,
			corev1.ConditionTrue,
			"",
			"Cluster data encryption has been initialized",
		)
	}); err != nil {
		return &reconcile.Result{}, err
	}

	return &reconcile.Result{}, nil
}

func (r *Reconciler) encryptData(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// TODO: spawn a Job so a restart of the seed-controller-manager does not re-run data encryption?
	return nil, nil
}

func (r *Reconciler) updateCluster(ctx context.Context, cluster *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster), opts ...ctrlruntimeclient.MergeFromOption) error {
	oldCluster := cluster.DeepCopy()
	modify(cluster)
	if reflect.DeepEqual(oldCluster, cluster) {
		return nil
	}

	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFromWithOptions(oldCluster, opts...))
}
