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

package usersshkeyssynchronizer

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "usersshkeys_synchronizer"

	UserSSHKeysClusterIDsCleanupFinalizer = "kubermatic.io/cleanup-usersshkeys-cluster-ids"
)

// Reconciler is a controller which is responsible for managing clusters
type Reconciler struct {
	ctx         context.Context
	client      ctrlruntimeclient.Client
	log         *zap.SugaredLogger
	workerName  string
	recorder    record.EventRecorder
	seedClients map[string]ctrlruntimeclient.Client
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
		return fmt.Errorf("failed to build worker-name selector: %v", err)
	}

	reconciler := &Reconciler{
		ctx:         ctx,
		log:         log.Named(ControllerName),
		workerName:  workerName,
		client:      mgr.GetClient(),
		recorder:    mgr.GetEventRecorderFor(ControllerName),
		seedClients: map[string]ctrlruntimeclient.Client{},
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	for seedName, seedManager := range seedManagers {
		reconciler.seedClients[seedName] = seedManager.GetClient()

		secretSource := &source.Kind{Type: &corev1.Secret{}}
		if err := secretSource.InjectCache(seedManager.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache into secretSource: %v", err)
		}
		if err := c.Watch(
			secretSource,
			controllerutil.EnqueueClusterForNamespacedObjectWithSeedName(seedManager.GetClient(), seedName, workerSelector),
			predicateutil.ByName(resources.UserSSHKeys),
		); err != nil {
			return fmt.Errorf("failed to establish watch for secrets in seed %s: %v", seedName, err)
		}

		clusterSource := &source.Kind{Type: &kubermaticv1.Cluster{}}
		if err := clusterSource.InjectCache(mgr.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache into clusterSource for seed %s: %v", seedName, err)
		}
		if err := c.Watch(
			clusterSource,
			controllerutil.EnqueueClusterScopedObjectWithSeedName(seedName),
			workerlabel.Predicates(workerName),
		); err != nil {
			return fmt.Errorf("failed to establish watch for clusters in seed %s: %v", seedName, err)
		}
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.UserSSHKey{}},
		enqueueAllClusters(reconciler.seedClients, workerSelector),
	); err != nil {
		return fmt.Errorf("failed to create watch for userSSHKey: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	err := r.reconcile(log, request)
	if controllerutil.IsCacheNotStarted(err) {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	if err != nil {
		log.Errorw("Reconciliation failed", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(log *zap.SugaredLogger, request reconcile.Request) error {
	seedClient, ok := r.seedClients[request.Namespace]
	if !ok {
		log.Errorw("Got request for seed we don't have a client for", "seed", request.Namespace)
		// The clients are inserted during controller initialization, so there is no point in retrying
		return nil
	}

	// find all clusters in this seed
	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(r.ctx, types.NamespacedName{Name: request.Name}, cluster); err != nil {
		if controllerutil.IsCacheNotStarted(err) {
			return err
		}

		if kubeapierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return nil
		}

		return fmt.Errorf("failed to get cluster %s from seed %s: %v", cluster.Name, request.Namespace, err)
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		log.Debugw(
			"Skipping because the cluster has a different worker name set",
			"cluster-worker-name", cluster.Labels[kubermaticv1.WorkerNameLabelKey],
		)
		return nil
	}

	if cluster.Spec.Pause {
		log.Debug("Skipping cluster reconciling because it was set to paused")
		return nil
	}

	userSSHKeys := &kubermaticv1.UserSSHKeyList{}
	if err := r.client.List(r.ctx, userSSHKeys); err != nil {
		return fmt.Errorf("failed to list userSSHKeys: %v", err)
	}

	if cluster.DeletionTimestamp != nil {
		if err := r.cleanupUserSSHKeys(userSSHKeys.Items, cluster.Name); err != nil {
			return fmt.Errorf("failed reconciling usersshkey: %v", err)
		}

		if kubernetes.HasFinalizer(cluster, UserSSHKeysClusterIDsCleanupFinalizer) {
			oldCluster := cluster.DeepCopy()
			kubernetes.RemoveFinalizer(cluster, UserSSHKeysClusterIDsCleanupFinalizer)
			if err := seedClient.Patch(r.ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
				return fmt.Errorf("failed removing %s finalizer: %v", UserSSHKeysClusterIDsCleanupFinalizer, err)
			}
		}
		return nil
	}

	keys := buildUserSSHKeysForCluster(cluster.Name, userSSHKeys)

	if err := reconciling.ReconcileSecrets(
		r.ctx,
		[]reconciling.NamedSecretCreatorGetter{updateUserSSHKeysSecrets(keys)},
		cluster.Status.NamespaceName,
		seedClient,
	); err != nil {
		return fmt.Errorf("failed to reconcile ssh key secret: %v", err)
	}

	oldCluster := cluster.DeepCopy()
	if !kubernetes.HasFinalizer(cluster, UserSSHKeysClusterIDsCleanupFinalizer) {
		kubernetes.AddFinalizer(cluster, UserSSHKeysClusterIDsCleanupFinalizer)
		if err := seedClient.Patch(r.ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return fmt.Errorf("failed adding %s finalizer: %v", UserSSHKeysClusterIDsCleanupFinalizer, err)
		}
	}

	return nil
}

func (r *Reconciler) cleanupUserSSHKeys(keys []kubermaticv1.UserSSHKey, clusterName string) error {
	for _, userSSHKey := range keys {
		userSSHKey.RemoveFromCluster(clusterName)
		if err := r.client.Update(r.ctx, &userSSHKey); err != nil {
			return fmt.Errorf("failed updating usersshkeys object: %v", err)
		}
	}

	return nil
}

func buildUserSSHKeysForCluster(clusterName string, list *kubermaticv1.UserSSHKeyList) []kubermaticv1.UserSSHKey {
	var clusterKeys []kubermaticv1.UserSSHKey
	for _, item := range list.Items {
		for _, clusterID := range item.Spec.Clusters {
			if clusterName == clusterID {
				clusterKeys = append(clusterKeys, item)
			}
		}
	}

	return clusterKeys
}

// enqueueAllClusters enqueues all clusters
func enqueueAllClusters(clients map[string]ctrlruntimeclient.Client, workerSelector labels.Selector) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		var requests []reconcile.Request

		listOpts := &ctrlruntimeclient.ListOptions{
			LabelSelector: workerSelector,
		}

		for seedName, client := range clients {
			clusterList := &kubermaticv1.ClusterList{}
			if err := client.List(context.Background(), clusterList, listOpts); err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to list Clusters in seed %s: %v", seedName, err))
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
	})}
}

// updateUserSSHKeysSecrets creates a secret in the seed cluster from the user ssh keys.
func updateUserSSHKeysSecrets(list []kubermaticv1.UserSSHKey) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.UserSSHKeys, func(existing *corev1.Secret) (secret *corev1.Secret, e error) {
			existing.Data = map[string][]byte{}

			for _, key := range list {
				existing.Data[key.Name] = []byte(key.Spec.PublicKey)
			}

			existing.Type = corev1.SecretTypeOpaque

			return existing, nil
		}
	}
}
