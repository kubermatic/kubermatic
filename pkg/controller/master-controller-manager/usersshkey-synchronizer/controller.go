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

package usersshkeysynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	controllerutil "k8c.io/kubermatic/v3/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	kuberneteshelper "k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/util/workerlabel"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
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
	ControllerName = "kkp-usersshkey-synchronizer"

	// UserSSHKeysClusterIDsCleanupFinalizer is the finalizer that is placed on a Cluster object
	// to indicate that the assigned SSH keys still need to be cleaned up.
	UserSSHKeysClusterIDsCleanupFinalizer = "kubermatic.k8c.io/cleanup-usersshkeys-cluster-ids"
)

// Reconciler is a controller which is responsible for synchronizing the
// assigned UserSSHKeys (on the master cluster) as Secrets into the seed
// clusters.
type Reconciler struct {
	masterClient ctrlruntimeclient.Client
	log          *zap.SugaredLogger
	workerName   string
	seedClients  kuberneteshelper.SeedClientMap
}

func Add(
	ctx context.Context,
	mgr manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &Reconciler{
		log:          log.Named(ControllerName),
		workerName:   workerName,
		masterClient: mgr.GetClient(),
		seedClients:  kuberneteshelper.SeedClientMap{},
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	for seedName, seedManager := range seedManagers {
		reconciler.seedClients[seedName] = seedManager.GetClient()

		secretSource := &source.Kind{Type: &corev1.Secret{}}
		if err := secretSource.InjectCache(seedManager.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache into secretSource: %w", err)
		}
		if err := c.Watch(
			secretSource,
			controllerutil.EnqueueClusterForNamespacedObjectWithSeedName(seedManager.GetClient(), seedName, workerSelector),
			predicateutil.ByName(resources.UserSSHKeys),
		); err != nil {
			return fmt.Errorf("failed to establish watch for secrets in seed %s: %w", seedName, err)
		}

		clusterSource := &source.Kind{Type: &kubermaticv1.Cluster{}}
		if err := clusterSource.InjectCache(mgr.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache into clusterSource for seed %s: %w", seedName, err)
		}
		if err := c.Watch(
			clusterSource,
			controllerutil.EnqueueClusterScopedObjectWithSeedName(seedName),
			workerlabel.Predicates(workerName),
		); err != nil {
			return fmt.Errorf("failed to establish watch for clusters in seed %s: %w", seedName, err)
		}
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.UserSSHKey{}},
		enqueueAllClusters(ctx, reconciler.seedClients, workerSelector),
	); err != nil {
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
	seedClient, ok := r.seedClients[request.Namespace]
	if !ok {
		log.Errorw("Got request for seed we don't have a client for", "seed", request.Namespace)
		// The clients are inserted during controller initialization, so there is no point in retrying
		return nil
	}

	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: request.Name}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return nil
		}

		return fmt.Errorf("failed to get cluster %s from seed %s: %w", cluster.Name, request.Namespace, err)
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
	if err := r.masterClient.List(ctx, userSSHKeys); err != nil {
		return fmt.Errorf("failed to list UserSSHKeys: %w", err)
	}

	if cluster.DeletionTimestamp != nil {
		if err := r.cleanupUserSSHKeys(ctx, userSSHKeys.Items, cluster.Name); err != nil {
			return fmt.Errorf("failed reconciling keys for a deleted cluster: %w", err)
		}

		return kubernetes.TryRemoveFinalizer(ctx, seedClient, cluster, UserSSHKeysClusterIDsCleanupFinalizer)
	}

	keys := buildUserSSHKeysForCluster(cluster.Name, userSSHKeys)

	if err := reconciling.ReconcileSecrets(
		ctx,
		[]reconciling.NamedSecretReconcilerFactory{updateUserSSHKeysSecrets(keys)},
		cluster.Status.NamespaceName,
		seedClient,
	); err != nil {
		return fmt.Errorf("failed to reconcile SSH key secret: %w", err)
	}

	if err := kubernetes.TryAddFinalizer(ctx, seedClient, cluster, UserSSHKeysClusterIDsCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	return nil
}

func (r *Reconciler) cleanupUserSSHKeys(ctx context.Context, keys []kubermaticv1.UserSSHKey, clusterName string) error {
	for _, userSSHKey := range keys {
		oldKey := userSSHKey.DeepCopy()
		userSSHKey.RemoveFromCluster(clusterName)
		if err := r.masterClient.Patch(ctx, &userSSHKey, ctrlruntimeclient.MergeFrom(oldKey)); err != nil {
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
func enqueueAllClusters(ctx context.Context, clients kuberneteshelper.SeedClientMap, workerSelector labels.Selector) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		listOpts := &ctrlruntimeclient.ListOptions{
			LabelSelector: workerSelector,
		}

		for seedName, client := range clients {
			clusterList := &kubermaticv1.ClusterList{}
			if err := client.List(ctx, clusterList, listOpts); err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to list Clusters in seed %s: %w", seedName, err))
				continue
			}
			for _, cluster := range clusterList.Items {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Namespace: seedName,
					Name:      cluster.Name,
				}})
			}
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
