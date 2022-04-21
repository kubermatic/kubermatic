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
	"errors"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

	seedGetter   provider.SeedGetter
	configGetter provider.KubermaticConfigurationGetter

	log                     *zap.SugaredLogger
	userClusterConnProvider userClusterConnectionProvider
	workerName              string
	recorder                record.EventRecorder

	overwriteRegistry string
	versions          kubermatic.Versions
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,

	numWorkers int,
	workerName string,

	userClusterConnProvider userClusterConnectionProvider,
	seedGetter provider.SeedGetter,
	configGetter provider.KubermaticConfigurationGetter,

	versions kubermatic.Versions,
	overwriteRegistry string,
) error {

	reconciler := &Reconciler{
		log:                     log.Named(ControllerName),
		Client:                  mgr.GetClient(),
		userClusterConnProvider: userClusterConnProvider,
		seedGetter:              seedGetter,
		configGetter:            configGetter,
		workerName:              workerName,

		recorder: mgr.GetEventRecorderFor(ControllerName),

		versions:          versions,
		overwriteRegistry: overwriteRegistry,
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
		encryptionEnabled := cluster.Spec.EncryptionConfiguration != nil && cluster.Spec.EncryptionConfiguration.Enabled
		encryptionActive := cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEncryptionInitialized, corev1.ConditionTrue)
		return cluster.Spec.Features[kubermaticv1.ClusterFeatureEncryptionAtRest] && (encryptionEnabled || encryptionActive)
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

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess,
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
		log.Debug("EncryptionInitialized is not set yet, setting initial encryption status and condition ...")
		return r.setInitializedCondition(ctx, cluster)
	}

	// this should never happen as the field should be initialized with the condition, but you never know ...
	if cluster.Status.Encryption == nil {
		return &reconcile.Result{}, errors.New("expected cluster.status.encryption to exist, but is nil")
	}

	switch cluster.Status.Encryption.Phase {
	case kubermaticv1.ClusterEncryptionPhasePending:
		isUpdated, err := isApiserverUpdated(ctx, r.Client, cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		if !isUpdated {
			log.Debug("kube-apiserver is not using updated EncryptionConfiguration yet, retrying in 10s ...")
			return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}

		keyHint, err := getActiveKey(ctx, r.Client, cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
			if c.Status.Encryption.ActiveKey != keyHint {
				// the active key as per the parsed EncryptionConfiguration has changed; we need to re-run encryption
				c.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseEncryptionNeeded
			} else {
				// EncryptionConfiguration was changed but the primary key did not change, so there is no need to re-run
				// encryption. We can skip right to ClusterEncryptionPhaseActive.
				c.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseActive
			}
		}); err != nil {
			return &reconcile.Result{}, err
		}

		return &reconcile.Result{}, nil

	case kubermaticv1.ClusterEncryptionPhaseEncryptionNeeded:
		key, err := getActiveKey(ctx, r.Client, cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		return r.encryptData(ctx, log, cluster, key)

	case kubermaticv1.ClusterEncryptionPhaseActive:
		// get a key hint as defined in the ClusterSpec to compare to the current status
		configuredKey, err := getConfiguredKey(cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		if cluster.Status.Encryption.ActiveKey != configuredKey {
			log.Debugf("configured key %q != %q, moving cluster to EncryptionPhase 'Pending'", configuredKey, cluster.Status.Encryption.ActiveKey)
			if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
				c.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhasePending
			}); err != nil {
				return &reconcile.Result{}, err
			}
		}
		return &reconcile.Result{}, nil

	case kubermaticv1.ClusterEncryptionPhaseFailed:
		// TODO: how to recover from a failed encryption? Can you even recover automatically?
		return &reconcile.Result{}, nil

	default:
		return &reconcile.Result{}, nil
	}
}

func (r *Reconciler) setInitializedCondition(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// set the encryption initialized condition (should only happen once on every cluster)
	if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
		if cluster.Status.Encryption == nil {
			cluster.Status.Encryption = &kubermaticv1.ClusterEncryptionStatus{
				Phase: kubermaticv1.ClusterEncryptionPhasePending,
			}
		}

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

func (r *Reconciler) updateCluster(ctx context.Context, cluster *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster), opts ...ctrlruntimeclient.MergeFromOption) error {
	oldCluster := cluster.DeepCopy()
	modify(cluster)
	if reflect.DeepEqual(oldCluster, cluster) {
		return nil
	}

	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFromWithOptions(oldCluster, opts...))
}

func (r *Reconciler) getClusterTemplateData(ctx context.Context, cluster *kubermaticv1.Cluster, seed *kubermaticv1.Seed, config *kubermaticv1.KubermaticConfiguration) (*resources.TemplateData, error) {
	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	return resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithClient(r).
		WithCluster(cluster).
		WithDatacenter(&datacenter).
		WithSeed(seed.DeepCopy()).
		WithKubermaticConfiguration(config.DeepCopy()).
		WithOverwriteRegistry(r.overwriteRegistry).
		Build(), nil
}
