/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package encryptionsecretsynchonizer

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName         = "kkp-encryption-secret-synchronizer"
	ClusterNameAnnotation  = "kubermatic.io/cluster-name"
	EncryptionSecretPrefix = "encryption-key-cluster-"

	// EncryptionSecretCleanupFinalizer is the finalizer that is placed on a Cluster object
	// to indicate that the assigned encryption secrets still need to be cleaned up.
	EncryptionSecretCleanupFinalizer = "kubermatic.k8c.io/cleanup-encryption-secrets"
)

// reconciler is a controller which is responsible for synchronizing the
// encryption secrets (in the master cluster) as Secrets into the seed
// clusters when clusters have encryption enabled.
type reconciler struct {
	masterClient ctrlruntimeclient.Client
	log          *zap.SugaredLogger
	recorder     events.EventRecorder
	workerName   string
	seedClients  kuberneteshelper.SeedClientMap
	namespace    string
}

func Add(
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	namespace string,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	r := &reconciler{
		log:          log.Named(ControllerName),
		workerName:   workerName,
		masterClient: masterManager.GetClient(),
		recorder:     masterManager.GetEventRecorder(ControllerName),
		seedClients:  kuberneteshelper.SeedClientMap{},
		namespace:    namespace,
	}

	bldr := builder.ControllerManagedBy(masterManager).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		Watches(&corev1.Secret{}, enqueueAllClustersForEncryptionSecret(r.seedClients, workerSelector, r.namespace))

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()

		bldr.WatchesRawSource(source.Kind(
			seedManager.GetCache(),
			&kubermaticv1.Cluster{},
			controllerutil.TypedEnqueueClusterScopedObjectWithSeedName[*kubermaticv1.Cluster](seedName),
			workerlabel.TypedPredicate[*kubermaticv1.Cluster](workerName),
		))
	}

	_, err = bldr.Build(r)
	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {
	seedClient, ok := r.seedClients[request.Namespace]
	if !ok {
		log.Error("Got request for seed we don't have a client for", "seed", request.Namespace)
		return nil
	}

	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: request.Name}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return nil
		}
		return fmt.Errorf("failed to get cluster %s from seed %s: %w", request.Name, request.Namespace, err)
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Skipping cluster reconciling because no namespace is set")
		return nil
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return nil
	}

	if cluster.Spec.Pause {
		log.Debug("Skipping cluster reconciling because it was set to paused")
		return nil
	}

	// Cleanup secrets on cluster deletion
	if cluster.DeletionTimestamp != nil {
		if err := r.cleanupEncryptionSecrets(ctx, log, cluster.Name); err != nil {
			return fmt.Errorf("failed to cleanup encryption secrets for deleted cluster: %w", err)
		}
		return kuberneteshelper.TryRemoveFinalizer(ctx, seedClient, cluster, EncryptionSecretCleanupFinalizer)
	}

	if !cluster.IsEncryptionEnabled() {
		if kuberneteshelper.HasFinalizer(cluster, EncryptionSecretCleanupFinalizer) {
			if err := r.cleanupEncryptionSecrets(ctx, log, cluster.Name); err != nil {
				return fmt.Errorf("failed to cleanup encryption secrets for cluster with disabled encryption: %w", err)
			}
		}
		return kuberneteshelper.TryRemoveFinalizer(ctx, seedClient, cluster, EncryptionSecretCleanupFinalizer)
	}

	// Get the encryption secret from kubermatic namespace
	secretName := EncryptionSecretPrefix + cluster.Name
	secret := &corev1.Secret{}
	if err := r.masterClient.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: r.namespace,
	}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Encryption secret not found in kubermatic namespace", "secret", secretName)
			return nil
		}
		return fmt.Errorf("failed to get encryption secret %s: %w", secretName, err)
	}

	// Verify the secret has the correct annotation
	if clusterName, exists := secret.Annotations[ClusterNameAnnotation]; !exists || clusterName != cluster.Name {
		log.Debug("Secret missing or incorrect cluster name annotation", "secret", secretName, "expectedCluster", cluster.Name)
		return nil
	}

	// Sync the secret to the cluster namespace
	clusterNamespace := cluster.Status.NamespaceName

	syncSecret := secret.DeepCopy()
	syncSecret.SetResourceVersion("")
	syncSecret.SetNamespace(clusterNamespace)

	if err := reconciling.ReconcileSecrets(
		ctx,
		[]reconciling.NamedSecretReconcilerFactory{secretReconcilerFactory(syncSecret)},
		clusterNamespace,
		seedClient,
	); err != nil {
		return fmt.Errorf("failed to reconcile encryption secret: %w", err)
	}

	// Add finalizer to ensure cleanup on cluster deletion
	if err := kuberneteshelper.TryAddFinalizer(ctx, seedClient, cluster, EncryptionSecretCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	return nil
}

func (r *reconciler) cleanupEncryptionSecrets(ctx context.Context, log *zap.SugaredLogger, clusterName string) error {
	secretName := EncryptionSecretPrefix + clusterName
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: r.namespace,
		},
	}

	if err := r.masterClient.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete encryption secret %s from kubermatic namespace: %w", secretName, err)
	}

	return r.seedClients.Each(ctx, log, func(seedName string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		cluster := &kubermaticv1.Cluster{}
		if err := seedClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		if cluster.Status.NamespaceName == "" {
			return nil
		}

		ucSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: cluster.Status.NamespaceName,
			},
		}

		if err := seedClient.Delete(ctx, ucSecret); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete encryption secret %s from UC namespace: %w", secretName, err)
		}

		return nil
	})
}

func secretReconcilerFactory(s *corev1.Secret) reconciling.NamedSecretReconcilerFactory {
	return func() (name string, create reconciling.SecretReconciler) {
		return s.Name, func(existing *corev1.Secret) (*corev1.Secret, error) {
			existing.Labels = s.Labels
			existing.Annotations = s.Annotations
			existing.Data = s.Data
			existing.Type = s.Type
			return existing, nil
		}
	}
}

// enqueueAllClustersForEncryptionSecret enqueues all clusters that have encryption enabled.
func enqueueAllClustersForEncryptionSecret(clients kuberneteshelper.SeedClientMap, workerSelector labels.Selector, kubermaticNamespace string) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		secret := obj.(*corev1.Secret)

		if !strings.HasPrefix(secret.Name, EncryptionSecretPrefix) {
			return nil
		}

		if secret.Namespace != kubermaticNamespace {
			return nil
		}

		clusterName, exists := secret.Annotations[ClusterNameAnnotation]
		if !exists {
			return nil
		}

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
				if cluster.Name == clusterName && cluster.IsEncryptionEnabled() {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: seedName,
							Name:      clusterName,
						},
					})
					break
				}
			}
		}

		return requests
	})
}
