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

package clusterusersshkeyscontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	controllerutil "k8c.io/kubermatic/v3/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/util/workerlabel"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this controller.
	ControllerName = "kkp-cluster-usersshkeys-controller"

	// UserSSHKeysClusterIDsCleanupFinalizer is the finalizer that is placed on a Cluster object
	// to indicate that the assigned SSH keys still need to be cleaned up.
	UserSSHKeysClusterIDsCleanupFinalizer = "kubermatic.k8c.io/cleanup-usersshkeys-cluster-ids"
)

// Reconciler is a controller which is responsible for synchronizing the
// assigned UserSSHKeys (on the master cluster) as Secrets into the seed
// clusters.
type Reconciler struct {
	seedClient ctrlruntimeclient.Client
	log        *zap.SugaredLogger
	workerName string
}

func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {
	reconciler := &Reconciler{
		log:        log.Named(ControllerName),
		workerName: workerName,
		seedClient: mgr.GetClient(),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	clusterEnqueuer := controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())

	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, clusterEnqueuer, predicateutil.ByName(resources.UserSSHKeys)); err != nil {
		return fmt.Errorf("failed to establish watch for secrets: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, clusterEnqueuer, workerlabel.Predicates(workerName)); err != nil {
		return fmt.Errorf("failed to establish watch for clusters: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.UserSSHKey{}}, enqueueAllClusters(ctx, mgr.GetClient())); err != nil {
		return fmt.Errorf("failed to create watch for userSSHKey: %w", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)
	if err != nil {
		log.Errorw("Reconciliation failed", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {
	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{Name: request.Name}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to get cluster: %w", err)
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Skipping cluster reconciling because no namespaceName was yet set")
		return nil
	}

	if cluster.Labels[workerlabel.LabelKey] != r.workerName {
		log.Debugw(
			"Skipping because the cluster has a different worker name set",
			"cluster-worker-name", cluster.Labels[workerlabel.LabelKey],
		)
		return nil
	}

	if cluster.Spec.Pause {
		log.Debug("Skipping cluster reconciling because it was set to paused")
		return nil
	}

	userSSHKeys := &kubermaticv1.UserSSHKeyList{}
	if err := r.seedClient.List(ctx, userSSHKeys); err != nil {
		return fmt.Errorf("failed to list UserSSHKeys: %w", err)
	}

	if cluster.DeletionTimestamp != nil {
		if err := r.cleanupUserSSHKeys(ctx, userSSHKeys.Items, cluster.Name); err != nil {
			return fmt.Errorf("failed reconciling keys for a deleted cluster: %w", err)
		}

		return kubernetes.TryRemoveFinalizer(ctx, r.seedClient, cluster, UserSSHKeysClusterIDsCleanupFinalizer)
	}

	if err := kubernetes.TryAddFinalizer(ctx, r.seedClient, cluster, UserSSHKeysClusterIDsCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	keys := buildUserSSHKeysForCluster(cluster.Name, userSSHKeys)

	if err := reconciling.ReconcileSecrets(
		ctx,
		[]reconciling.NamedSecretReconcilerFactory{updateUserSSHKeysSecrets(keys)},
		cluster.Status.NamespaceName,
		r.seedClient,
	); err != nil {
		return fmt.Errorf("failed to reconcile SSH key secret: %w", err)
	}

	return nil
}

func (r *Reconciler) cleanupUserSSHKeys(ctx context.Context, keys []kubermaticv1.UserSSHKey, clusterName string) error {
	for _, userSSHKey := range keys {
		oldKey := userSSHKey.DeepCopy()
		userSSHKey.RemoveFromCluster(clusterName)
		if err := r.seedClient.Patch(ctx, &userSSHKey, ctrlruntimeclient.MergeFrom(oldKey)); err != nil {
			return fmt.Errorf("failed updating UserSSHKey object: %w", err)
		}
	}

	return nil
}

func buildUserSSHKeysForCluster(clusterName string, keys *kubermaticv1.UserSSHKeyList) []kubermaticv1.UserSSHKey {
	var clusterKeys []kubermaticv1.UserSSHKey
	for _, key := range keys.Items {
		if key.IsUsedByCluster(clusterName) {
			clusterKeys = append(clusterKeys, key)
		}
	}

	return clusterKeys
}

// enqueueAllClusters enqueues all clusters.
func enqueueAllClusters(ctx context.Context, client ctrlruntimeclient.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		sshKey, ok := a.(*kubermaticv1.UserSSHKey)
		if !ok {
			return nil
		}

		requests := []reconcile.Request{}

		for _, clusterName := range sshKey.Spec.Clusters {
			cluster := &kubermaticv1.Cluster{}
			key := types.NamespacedName{Name: clusterName}

			if err := client.Get(ctx, key, cluster); err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to get Cluster: %w", err))
				continue
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(cluster),
			})
		}

		return requests
	})
}

// updateUserSSHKeysSecrets creates a secret in the seed cluster from the user ssh keys.
func updateUserSSHKeysSecrets(keys []kubermaticv1.UserSSHKey) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.UserSSHKeys, func(existing *corev1.Secret) (secret *corev1.Secret, e error) {
			existing.Data = map[string][]byte{}

			for _, key := range keys {
				existing.Data[key.Name] = []byte(key.Spec.PublicKey)
			}

			existing.Type = corev1.SecretTypeOpaque

			return existing, nil
		}
	}
}
