/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package clustertemplatesynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller syncs the kubermatic cluster templates on the master cluster to the seed clusters.
	ControllerName = "kkp-cluster-template-synchronizer"

	// cleanupFinalizer indicates that synced cluster template on seed clusters need cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-cluster-template"
)

type reconciler struct {
	log          *zap.SugaredLogger
	seedsGetter  provider.SeedsGetter
	masterClient ctrlruntimeclient.Client
	seedClients  kuberneteshelper.SeedClientMap
	recorder     events.EventRecorder
}

func Add(
	masterMgr manager.Manager,
	seedsGetter provider.SeedsGetter,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
) error {
	log = log.Named(ControllerName)
	r := &reconciler{
		log:          log,
		seedsGetter:  seedsGetter,
		masterClient: masterMgr.GetClient(),
		seedClients:  kuberneteshelper.SeedClientMap{},
		recorder:     masterMgr.GetEventRecorder(ControllerName),
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	_, err := builder.ControllerManagedBy(masterMgr).
		Named(ControllerName).
		For(&kubermaticv1.ClusterTemplate{}).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("template", request.Name)
	log.Debug("Processing")

	clusterTemplate := &kubermaticv1.ClusterTemplate{}
	if err := r.masterClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: request.Name}, clusterTemplate); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, log, clusterTemplate)
	if err != nil {
		r.recorder.Eventf(clusterTemplate, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, clusterTemplate *kubermaticv1.ClusterTemplate) error {
	// handling deletion
	if !clusterTemplate.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, clusterTemplate); err != nil {
			return fmt.Errorf("handling deletion of cluster template: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, clusterTemplate, cleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	clusterTemplateReconcilerFactories := []reconciling.NamedClusterTemplateReconcilerFactory{
		clusterTemplateReconcilerFactory(clusterTemplate),
	}

	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		seedTpl := &kubermaticv1.ClusterTemplate{}
		if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(clusterTemplate), seedTpl); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch ClusterTemplate on seed cluster: %w", err)
		}

		// see project-synchronizer's syncAllSeeds comment
		if seedTpl.UID != "" && seedTpl.UID == clusterTemplate.UID {
			return nil
		}

		return reconciling.ReconcileClusterTemplates(ctx, clusterTemplateReconcilerFactories, "", seedClient)
	})
	if err != nil {
		return fmt.Errorf("reconciled cluster template: %s: %w", clusterTemplate.Name, err)
	}
	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, template *kubermaticv1.ClusterTemplate) error {
	if kuberneteshelper.HasFinalizer(template, cleanupFinalizer) {
		if err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, _ *zap.SugaredLogger) error {
			err := seedClient.Delete(ctx, &kubermaticv1.ClusterTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: template.Name,
				},
			})

			return ctrlruntimeclient.IgnoreNotFound(err)
		}); err != nil {
			return err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, template, cleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove cluster template finalizer %s: %w", template.Name, err)
		}
	}

	if kuberneteshelper.HasFinalizer(template, kubermaticv1.CredentialsSecretsCleanupFinalizer) {
		if err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, _ *zap.SugaredLogger) error {
			err := seedClient.Delete(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      template.Credential,
					Namespace: resources.KubermaticNamespace,
				},
			})
			return ctrlruntimeclient.IgnoreNotFound(err)
		}); err != nil {
			return err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, template, kubermaticv1.CredentialsSecretsCleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove credential secret finalizer %s: %w", template.Name, err)
		}
	}

	return nil
}

func clusterTemplateReconcilerFactory(template *kubermaticv1.ClusterTemplate) reconciling.NamedClusterTemplateReconcilerFactory {
	return func() (string, reconciling.ClusterTemplateReconciler) {
		return template.Name, func(c *kubermaticv1.ClusterTemplate) (*kubermaticv1.ClusterTemplate, error) {
			c.Name = template.Name
			c.Spec = template.Spec
			c.Labels = template.Labels
			c.Annotations = template.Annotations
			c.InheritedClusterLabels = template.InheritedClusterLabels
			c.Credential = template.Credential
			return c, nil
		}
	}
}
