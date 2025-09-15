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
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName         = "kkp-encryption-secret-synchronizer"
	ClusterNameAnnotation  = "kubermatic.io/cluster-name"
	EncryptionSecretPrefix = "encryption-key-cluster-"
)

type reconciler struct {
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	masterClient ctrlruntimeclient.Client
	namespace    string
	seedClients  kuberneteshelper.SeedClientMap
}

func Add(
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	namespace string,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	r := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     masterManager.GetEventRecorderFor(ControllerName),
		masterClient: masterManager.GetClient(),
		seedClients:  kuberneteshelper.SeedClientMap{},
		namespace:    namespace,
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	_, err := builder.ControllerManagedBy(masterManager).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(
			&corev1.Secret{},
			builder.WithPredicates(
				predicate.Factory(func(o ctrlruntimeclient.Object) bool {
					secret := o.(*corev1.Secret)
					if !strings.HasPrefix(secret.Name, EncryptionSecretPrefix) {
						return false
					}

					return metav1.HasAnnotation(secret.ObjectMeta, ClusterNameAnnotation)
				}),
				predicate.ByNamespace(r.namespace),
			),
		).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("secret", request)
	log.Debug("Processing")

	secret := &corev1.Secret{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}

		// handling deletion
		delSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: request.Name, Namespace: r.namespace}}
		if err := r.handleDeletion(ctx, log, delSecret); err != nil {
			err = fmt.Errorf("failed to delete secret: %w", err)

			log.Error("ReconcilingError", zap.Error(err))
			r.recorder.Event(delSecret, corev1.EventTypeWarning, "ReconcilingError", err.Error())

			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	return r.reconcile(ctx, log, secret)
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, secret *corev1.Secret) (reconcile.Result, error) {
	clusterName, exists := secret.Annotations[ClusterNameAnnotation]
	if !exists {
		return reconcile.Result{}, fmt.Errorf("secret %s missing cluster name annotation %s", secret.Name, ClusterNameAnnotation)
	}

	cluster, seedName, err := r.findTargetCluster(ctx, clusterName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Debug("Cluster not found, secret will be synced when cluster is created", "cluster", clusterName)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Wait for cluster to have a namespace before syncing the secret
	if cluster.Status.NamespaceName == "" {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if !cluster.IsEncryptionEnabled() {
		return reconcile.Result{}, nil
	}

	seedClient, exists := r.seedClients[seedName]
	if !exists {
		return reconcile.Result{}, fmt.Errorf("seed client not found for %s", seedName)
	}

	clusterNamespace := cluster.Status.NamespaceName
	syncSecret := secret.DeepCopy()
	syncSecret.SetResourceVersion("")
	syncSecret.SetNamespace(clusterNamespace)

	namedSecretReconcilerFactory := []reconciling.NamedSecretReconcilerFactory{
		secretReconcilerFactory(syncSecret),
	}

	if err := reconciling.ReconcileSecrets(ctx, namedSecretReconcilerFactory, clusterNamespace, seedClient); err != nil {
		return reconcile.Result{}, fmt.Errorf("reconciling encryption secret %s failed: %w", syncSecret.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) findTargetCluster(ctx context.Context, clusterName string) (*kubermaticv1.Cluster, string, error) {
	var targetCluster *kubermaticv1.Cluster
	var targetSeedName string

	err := r.seedClients.Each(ctx, r.log, func(seedName string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		cluster := &kubermaticv1.Cluster{}
		err := seedClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster)
		if err == nil {
			targetCluster = cluster
			targetSeedName = seedName
			return nil
		} else if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, "", err
	}

	if targetCluster == nil {
		return nil, "", fmt.Errorf("cluster %s not found in any seed", clusterName)
	}

	return targetCluster, targetSeedName, nil
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

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, secret *corev1.Secret) error {
	if !strings.HasPrefix(secret.Name, EncryptionSecretPrefix) {
		return nil
	}

	clusterName, exists := secret.Annotations[ClusterNameAnnotation]
	if !exists {
		return nil
	}

	return r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
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

		err := seedClient.Delete(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.Name,
				Namespace: cluster.Status.NamespaceName,
			},
		})

		if apierrors.IsNotFound(err) {
			return nil
		}

		return err
	})
}
