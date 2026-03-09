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

package applicationsecretsynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName       = "kkp-application-secret-synchronizer"
	SecretTypeAnnotation = "apps.kubermatic.k8c.io/secret-type"
)

type reconciler struct {
	log          *zap.SugaredLogger
	recorder     events.EventRecorder
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
		recorder:     masterManager.GetEventRecorder(ControllerName),
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
				predicate.ByAnnotation(SecretTypeAnnotation, "", false),
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

			log.Errorw("ReconcilingError", zap.Error(err))
			r.recorder.Eventf(delSecret, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())

			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	err := r.reconcile(ctx, log, secret)
	if err != nil {
		r.recorder.Eventf(secret, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, secret *corev1.Secret) error {
	seedsecret := secret.DeepCopy()
	seedsecret.SetResourceVersion("")

	namedSecretReconcilerFactory := []reconciling.NamedSecretReconcilerFactory{
		secretReconcilerFactory(seedsecret),
	}
	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		seedSecret := &corev1.Secret{}
		if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(seedsecret), seedSecret); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch Secret on seed cluster: %w", err)
		}

		// see project-synchronizer's syncAllSeeds comment
		if seedSecret.UID != "" && seedSecret.UID == seedsecret.UID {
			return nil
		}

		return reconciling.ReconcileSecrets(ctx, namedSecretReconcilerFactory, r.namespace, seedClient)
	})
	if err != nil {
		return fmt.Errorf("reconciling secret %s failed: %w", seedsecret.Name, err)
	}

	return nil
}

func secretReconcilerFactory(s *corev1.Secret) reconciling.NamedSecretReconcilerFactory {
	return func() (name string, create reconciling.SecretReconciler) {
		return s.Name, func(existing *corev1.Secret) (*corev1.Secret, error) {
			existing.Labels = s.Labels
			existing.Annotations = s.Annotations
			existing.Data = s.Data
			return existing, nil
		}
	}
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, secret *corev1.Secret) error {
	return r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		seedSecret := &corev1.Secret{}
		err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(secret), seedSecret)
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Debug("Secret already deleted")
				return nil
			}
			return err
		}

		return seedClient.Delete(ctx, seedSecret)
	})
}
