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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
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
	// This controller syncs the kubermatic cluster templates on the master cluster to the seed clusters.
	ControllerName = "kkp-cluster-template-synchronizer"
)

type reconciler struct {
	log          *zap.SugaredLogger
	masterClient ctrlruntimeclient.Client
	seedClients  map[string]ctrlruntimeclient.Client
	recorder     record.EventRecorder
}

func Add(
	masterMgr manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
) error {
	log = log.Named(ControllerName)
	r := &reconciler{
		log:          log,
		masterClient: masterMgr.GetClient(),
		seedClients:  map[string]ctrlruntimeclient.Client{},
		recorder:     masterMgr.GetEventRecorderFor(ControllerName),
	}

	c, err := controller.New(ControllerName, masterMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	// Watch for changes to ClusterTemplates
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.ClusterTemplate{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch cluster templates: %w", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {
	clusterTemplate := &kubermaticv1.ClusterTemplate{}
	if err := r.masterClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: request.Name}, clusterTemplate); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	// handling deletion
	if !clusterTemplate.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, clusterTemplate); err != nil {
			return fmt.Errorf("handling deletion of cluster template: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, clusterTemplate, apiv1.ClusterTemplateSeedCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	clusterTemplateCreatorGetters := []reconciling.NamedKubermaticV1ClusterTemplateCreatorGetter{
		clusterTemplateCreatorGetter(clusterTemplate),
	}

	err := r.syncAllSeeds(log, clusterTemplate, func(seedClient ctrlruntimeclient.Client, template *kubermaticv1.ClusterTemplate) error {
		seedTpl := &kubermaticv1.ClusterTemplate{}
		if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(template), seedTpl); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch ClusterTemplate on seed cluster: %w", err)
		}

		// see project-synchronizer's syncAllSeeds comment
		if seedTpl.UID != "" && seedTpl.UID == template.UID {
			return nil
		}

		return reconciling.ReconcileKubermaticV1ClusterTemplates(ctx, clusterTemplateCreatorGetters, "", seedClient)
	})
	if err != nil {
		r.recorder.Eventf(clusterTemplate, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return fmt.Errorf("reconciled cluster template: %s: %w", clusterTemplate.Name, err)
	}
	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, template *kubermaticv1.ClusterTemplate) error {
	if kuberneteshelper.HasFinalizer(template, apiv1.ClusterTemplateSeedCleanupFinalizer) {
		if err := r.syncAllSeeds(log, template, func(seedClient ctrlruntimeclient.Client, template *kubermaticv1.ClusterTemplate) error {
			err := seedClient.Delete(ctx, &kubermaticv1.ClusterTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: template.Name,
				},
			})

			return ctrlruntimeclient.IgnoreNotFound(err)
		}); err != nil {
			return err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, template, apiv1.ClusterTemplateSeedCleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove cluster template finalizer %s: %w", template.Name, err)
		}
	}

	if kuberneteshelper.HasFinalizer(template, apiv1.CredentialsSecretsCleanupFinalizer) {
		if err := r.syncAllSeeds(log, template, func(seedClient ctrlruntimeclient.Client, template *kubermaticv1.ClusterTemplate) error {
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

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, template, apiv1.CredentialsSecretsCleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove credential secret finalizer %s: %w", template.Name, err)
		}
	}

	return nil
}

func (r *reconciler) syncAllSeeds(log *zap.SugaredLogger, template *kubermaticv1.ClusterTemplate, action func(seedClient ctrlruntimeclient.Client, template *kubermaticv1.ClusterTemplate) error) error {
	for seedName, seedClient := range r.seedClients {
		log := log.With("seed", seedName)

		log.Debug("Reconciling cluster template with seed")

		err := action(seedClient, template)
		if err != nil {
			return fmt.Errorf("failed syncing cluster template %s for seed %s: %w", template.Name, seedName, err)
		}
		log.Debug("Reconciled cluster template with seed")
	}
	return nil
}

func clusterTemplateCreatorGetter(template *kubermaticv1.ClusterTemplate) reconciling.NamedKubermaticV1ClusterTemplateCreatorGetter {
	return func() (string, reconciling.KubermaticV1ClusterTemplateCreator) {
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
