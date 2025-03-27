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

package applicationsecretclustercontroller

import (
	"context"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	applicationsecretsynchronizer "k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/application-secret-synchronizer"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName  = "kkp-application-secret-cluster-controller"
	clusterPauseKey = ".spec.pause"
	isAppSecretKey  = "isAppSecretKey"

	// applicationSecretCleanupFinalizer indicates that secret synced from kubermatic namespace to cluster namespace need cleanup.
	applicationSecretCleanupFinalizer = "kubermatic.k8c.io/cleanup-application-secret"
)

type reconciler struct {
	client              ctrlruntimeclient.Client
	log                 *zap.SugaredLogger
	recorder            record.EventRecorder
	workerName          string
	namespace           string
	workerlabelSelector labels.Selector
}

func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	namespace string,
) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &kubermaticv1.Cluster{}, clusterPauseKey, func(rawObj ctrlruntimeclient.Object) []string {
		cluster := rawObj.(*kubermaticv1.Cluster)
		return []string{strconv.FormatBool(cluster.Spec.Pause)}
	}); err != nil {
		return fmt.Errorf("failed to add index on cluster.Spec.Pause: %w", err)
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.Secret{}, isAppSecretKey, func(rawObj ctrlruntimeclient.Object) []string {
		secret := rawObj.(*corev1.Secret)
		if secret.Annotations == nil {
			return nil
		}
		_, isAppSecret := secret.Annotations[applicationsecretsynchronizer.SecretTypeAnnotation]
		return []string{strconv.FormatBool(isAppSecret)}
	}); err != nil {
		return fmt.Errorf("failed to add index on Secret.metadata.annotation: %w", err)
	}

	workerlabelSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return err
	}

	r := &reconciler{
		log:                 log.Named(ControllerName),
		client:              mgr.GetClient(),
		workerName:          workerName,
		recorder:            mgr.GetEventRecorderFor(ControllerName),
		namespace:           namespace,
		workerlabelSelector: workerlabelSelector,
	}

	_, err = builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&corev1.Secret{}, builder.WithPredicates(predicateutil.ByAnnotation(applicationsecretsynchronizer.SecretTypeAnnotation, "", false), predicateutil.ByNamespace(r.namespace))).
		Watches(&kubermaticv1.Cluster{}, handler.EnqueueRequestsFromMapFunc(enqueueSecret(r.client, r.namespace)), builder.WithPredicates(workerlabel.Predicate(workerName), noDeleteEventPredicate())).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	secret := &corev1.Secret{}

	if err := r.client.Get(ctx, request.NamespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("secret not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get secret: %w", err)
	}

	err := r.reconcile(ctx, log, secret)
	if err != nil {
		r.recorder.Event(secret, corev1.EventTypeWarning, "SecretReconcileFailed", err.Error())
	}

	log.Debug("Processed")
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, secret *corev1.Secret) error {
	// handling deletion
	if !secret.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, secret); err != nil {
			return fmt.Errorf("handling deletion of secret: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.client, secret, applicationSecretCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	creators := []reconciling.NamedSecretReconcilerFactory{
		secretReconciler(secret),
	}
	if err := r.syncAllClusterNs(ctx, log, func(client ctrlruntimeclient.Client, clusterNamespace string) error {
		return reconciling.ReconcileSecrets(ctx, creators, clusterNamespace, r.client)
	}); err != nil {
		return err
	}

	return nil
}

func (r *reconciler) syncAllClusterNs(ctx context.Context, log *zap.SugaredLogger, action func(client ctrlruntimeclient.Client, clusterNamespace string) error) error {
	clusters := &kubermaticv1.ClusterList{}

	if err := r.client.List(ctx, clusters, &ctrlruntimeclient.ListOptions{LabelSelector: r.workerlabelSelector}, ctrlruntimeclient.MatchingFields{clusterPauseKey: "false"}); err != nil {
		return fmt.Errorf("can not list cluster: %w", err)
	}

	for _, cluster := range clusters.Items {
		log := log.With("cluster", cluster.Name)
		log.Debug("Reconciling secret with cluster")

		if !cluster.DeletionTimestamp.IsZero() {
			log.Debug("cluster deletion in progress, skipping")
			continue
		}

		if cluster.Status.NamespaceName == "" {
			log.Debug("cluster has no namespace name yet, skipping")
			continue
		}
		err := action(r.client, cluster.Status.NamespaceName)
		if err != nil {
			return fmt.Errorf("failed syncing to sync secret with cluster: %w", err)
		}
		log.Debug("Reconciled secret with cluster")
	}
	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, secret *corev1.Secret) error {
	if kuberneteshelper.HasFinalizer(secret, applicationSecretCleanupFinalizer) {
		if err := r.syncAllClusterNs(ctx, log, func(client ctrlruntimeclient.Client, clusterNamespace string) error {
			err := client.Delete(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret.Name,
					Namespace: clusterNamespace,
				},
			})

			return ctrlruntimeclient.IgnoreNotFound(err)
		}); err != nil {
			return err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.client, secret, applicationSecretCleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove secret finalizer: %w", err)
		}
	}
	return nil
}

func secretReconciler(s *corev1.Secret) reconciling.NamedSecretReconcilerFactory {
	return func() (name string, create reconciling.SecretReconciler) {
		return s.Name, func(existing *corev1.Secret) (*corev1.Secret, error) {
			existing.Data = s.Data
			existing.Labels = s.Labels
			existing.Annotations = s.Annotations
			return existing, nil
		}
	}
}

func enqueueSecret(client ctrlruntimeclient.Client, namespace string) func(context.Context, ctrlruntimeclient.Object) []reconcile.Request {
	return func(ctx context.Context, _ ctrlruntimeclient.Object) []reconcile.Request {
		secretList := &corev1.SecretList{}
		if err := client.List(ctx, secretList, &ctrlruntimeclient.ListOptions{Namespace: namespace}, ctrlruntimeclient.MatchingFields{isAppSecretKey: "true"}); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list secret: %w", err))
			return []reconcile.Request{}
		}
		var res []reconcile.Request
		for _, secret := range secretList.Items {
			res = append(res, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}})
		}
		return res
	}
}

func noDeleteEventPredicate() predicate.Funcs {
	return predicate.Funcs{
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
	}
}
