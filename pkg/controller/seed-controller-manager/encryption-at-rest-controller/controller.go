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

package encryptionatrestcontroller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	encryptionresources "k8c.io/kubermatic/v2/pkg/resources/encryption"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-encryption-at-rest-controller"
	EARKeyLength   = 32
)

// userClusterConnectionProvider offers functions to retrieve clients for the given user clusters.
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

	enqueueCluster := controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
			cluster := o.(*kubermaticv1.Cluster)
			return cluster.IsEncryptionEnabled() || cluster.IsEncryptionActive()
		}))).
		Watches(&corev1.Secret{}, enqueueCluster, builder.WithPredicates(predicateutil.ByName(resources.EncryptionConfigurationSecretName))).
		Watches(&appsv1.Deployment{}, enqueueCluster, builder.WithPredicates(predicateutil.ByName(resources.ApiserverDeploymentName))).
		Watches(&batchv1.Job{}, enqueueCluster, builder.WithPredicates(predicateutil.ByLabel(resources.AppLabelKey, encryptionresources.AppLabelValue))).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// replicate the predicate from above to make sure that reconcile loops triggered by apiserver and secret
	// do not run unexpected reconciles.
	if !cluster.IsEncryptionEnabled() && !cluster.IsEncryptionActive() {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := controllerutil.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionEncryptionControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
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

func getSecretKeyValue(ctx context.Context, client ctrlruntimeclient.Client, ref *corev1.SecretKeySelector, namespace string) ([]byte, error) {
	secret := corev1.Secret{}
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{
		Name:      ref.Name,
		Namespace: namespace,
	}, &secret); err != nil {
		return nil, err
	}

	val, ok := secret.Data[ref.Key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret", ref.Key)
	}

	return val, nil
}

// validateKeyLength base64 decodes key and checks length.
func validateKeyLength(key string) error {
	data, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	if len(data) != EARKeyLength {
		return fmt.Errorf("key length should be 32 it is %d", len(data))
	}
	return nil
}

func hasSecretKeyRef(cluster *kubermaticv1.Cluster) bool {
	if cluster.Spec.EncryptionConfiguration == nil {
		return false
	}

	if cluster.Spec.EncryptionConfiguration.Secretbox == nil {
		return false
	}

	for _, key := range cluster.Spec.EncryptionConfiguration.Secretbox.Keys {
		if key.SecretRef != nil {
			return true
		}
	}

	return false
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// reconcile until encryption is successfully initialized
	if cluster.IsEncryptionEnabled() && !cluster.IsEncryptionActive() {
		// validate secretRef before going into Pending phase
		result, err := r.validateSecretRef(ctx, cluster)
		if err != nil {
			return result, err
		}

		log.Debug("EncryptionInitialized is not set yet, setting initial encryption status and condition ...")
		return r.setInitializedCondition(ctx, cluster)
	}

	// this should never happen as the field should be initialized with the condition, but you never know ...
	if cluster.Status.Encryption == nil {
		return &reconcile.Result{}, errors.New("expected cluster.status.encryption to exist, but is nil")
	}

	switch cluster.Status.Encryption.Phase {
	case kubermaticv1.ClusterEncryptionPhasePending:
		isUpdated, err := isApiserverUpdated(ctx, r, cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		if !isUpdated {
			log.Debug("kube-apiserver is not using updated EncryptionConfiguration yet, retrying in 10s ...")
			return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}

		keyHint, resourceList, err := getActiveConfiguration(ctx, r, cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		if err := controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
			if c.Status.Encryption.ActiveKey != keyHint || !isEqualSlice(c.Status.Encryption.EncryptedResources, resourceList) {
				// the active key as per the parsed EncryptionConfiguration has changed; we need to re-run encryption
				c.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseEncryptionNeeded
			} else {
				// EncryptionConfiguration was changed but the primary configuration did not change, so there is no need to re-run
				// encryption. We can skip right to ClusterEncryptionPhaseActive.
				c.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhaseActive
			}
		}); err != nil {
			return &reconcile.Result{}, err
		}

		return &reconcile.Result{}, nil

	case kubermaticv1.ClusterEncryptionPhaseEncryptionNeeded:
		return r.encryptData(ctx, log, cluster)

	case kubermaticv1.ClusterEncryptionPhaseActive:
		// get a key hint as defined in the ClusterSpec to compare to the current status.
		configuredKey, err := getConfiguredKey(cluster)
		if err != nil {
			return &reconcile.Result{}, err
		}

		if cluster.Status.Encryption.ActiveKey != configuredKey {
			// validate secretRef before going into Pending phase
			result, err := r.validateSecretRef(ctx, cluster)
			if err != nil {
				return result, err
			}

			log.Debugf("configured key %q != %q, moving cluster to EncryptionPhase 'Pending'", configuredKey, cluster.Status.Encryption.ActiveKey)
			if err := controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
				c.Status.Encryption.Phase = kubermaticv1.ClusterEncryptionPhasePending
			}); err != nil {
				return &reconcile.Result{}, err
			}
		}

		// encryption is set to "identity", thus secrets are unencrypted, and encryption is longer wished.
		// This means we can fully reset the encryption status
		if cluster.Status.Encryption.ActiveKey == encryptionresources.IdentityKey && !cluster.IsEncryptionEnabled() {
			if err := controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
				c.Status.Encryption = nil
				controllerutil.SetClusterCondition(
					cluster,
					r.versions,
					kubermaticv1.ClusterConditionEncryptionInitialized,
					corev1.ConditionFalse,
					"",
					"Cluster data encryption has been removed",
				)
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

func (r *Reconciler) validateSecretRef(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if !hasSecretKeyRef(cluster) {
		return nil, nil
	}

	for _, key := range cluster.Spec.EncryptionConfiguration.Secretbox.Keys {
		if key.SecretRef != nil {
			v, err := getSecretKeyValue(ctx, r, key.SecretRef, fmt.Sprintf("cluster-%s", cluster.Name))
			if err != nil {
				return &reconcile.Result{}, err
			} else {
				if err := validateKeyLength(string(v)); err != nil {
					return &reconcile.Result{}, fmt.Errorf("%s->Secret:%s->Key:%s: %w", key.Name, key.SecretRef.Name, key.SecretRef.Key, err)
				}
			}
		}
	}

	return nil, nil
}

func (r *Reconciler) setInitializedCondition(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// set the encryption initialized condition (should only happen once on every cluster)
	if err := controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		if cluster.Status.Encryption == nil {
			cluster.Status.Encryption = &kubermaticv1.ClusterEncryptionStatus{
				Phase:              kubermaticv1.ClusterEncryptionPhasePending,
				EncryptedResources: []string{},
			}
		}

		controllerutil.SetClusterCondition(
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
