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
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName       = "kkp-application-secret-synchronizer"
	secretTypeAnnotation = "apps.kubermatic.k8c.io/secret-type"
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

	c, err := controller.New(ControllerName, masterManager, controller.Options{Reconciler: r, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}, predicate.ByAnnotation(secretTypeAnnotation, "", false), predicate.ByNamespace(r.namespace)); err != nil {
		return fmt.Errorf("failed to create watch for secrets: %w", err)
	}

	return nil
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
			r.recorder.Event(delSecret, corev1.EventTypeWarning, "ReconcilingError", err.Error())

			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	err := r.reconcile(ctx, log, secret)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
		r.recorder.Event(secret, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, secret *corev1.Secret) error {
	seedsecret := secret.DeepCopy()
	seedsecret.SetResourceVersion("")

	namedSecretCreatorGetter := []reconciling.NamedSecretCreatorGetter{
		secretCreator(seedsecret),
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

		return reconciling.ReconcileSecrets(ctx, namedSecretCreatorGetter, r.namespace, seedClient)
	})
	if err != nil {
		return fmt.Errorf("reconciling secret %s failed: %w", seedsecret.Name, err)
	}

	return nil
}

func secretCreator(s *corev1.Secret) reconciling.NamedSecretCreatorGetter {
	return func() (name string, create reconciling.SecretCreator) {
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
