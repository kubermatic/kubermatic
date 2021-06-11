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
	"time"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller syncs the kubermatic cluster templates on the master cluster to the seed clusters.
	ControllerName = "cluster_template_syncing_controller"
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
	log *zap.SugaredLogger) error {

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
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	// Watch for changes to ClusterTemplates
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.ClusterTemplate{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch cluster templates: %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {

	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)
	if controllerutil.IsCacheNotStarted(err) {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {
	clusterTemplate := &kubermaticv1.ClusterTemplate{}
	if err := r.masterClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: request.Name}, clusterTemplate); err != nil {
		if controllerutil.IsCacheNotStarted(err) {
			return err
		}
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	// handling deletion
	if !clusterTemplate.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, clusterTemplate); err != nil {
			return fmt.Errorf("handling deletion of cluster template: %v", err)
		}
		return nil
	}

	if !kuberneteshelper.HasFinalizer(clusterTemplate, kubermaticapiv1.ClusterTemplateSeedCleanupFinalizer) {
		kuberneteshelper.AddFinalizer(clusterTemplate, kubermaticapiv1.ClusterTemplateSeedCleanupFinalizer)
		if err := r.masterClient.Update(ctx, clusterTemplate); err != nil {
			return fmt.Errorf("failed to add finalizer: %v", err)
		}
	}

	clusterTemplateCreatorGetters := []reconciling.NamedKubermaticV1ClusterTemplateCreatorGetter{
		clusterTemplateCreatorGetter(clusterTemplate),
	}

	err := r.syncAllSeeds(log, clusterTemplate, func(seedClient ctrlruntimeclient.Client, template *kubermaticv1.ClusterTemplate) error {
		return reconciling.ReconcileKubermaticV1ClusterTemplates(ctx, clusterTemplateCreatorGetters, "", seedClient)
	})
	if err != nil {
		r.recorder.Eventf(clusterTemplate, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return fmt.Errorf("reconciled cluster template: %s: %v", clusterTemplate.Name, err)
	}
	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, template *kubermaticv1.ClusterTemplate) error {

	// if finalizer not set to master ClusterTemplate then return
	if !kuberneteshelper.HasFinalizer(template, kubermaticapiv1.ClusterTemplateSeedCleanupFinalizer) {
		return nil
	}

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

	if kuberneteshelper.HasFinalizer(template, kubermaticapiv1.ClusterTemplateSeedCleanupFinalizer) {
		kuberneteshelper.RemoveFinalizer(template, kubermaticapiv1.ClusterTemplateSeedCleanupFinalizer)
		if err := r.masterClient.Update(ctx, template); err != nil {
			return fmt.Errorf("failed to remove cluster template finalizer %s: %w", template.Name, err)
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

			return c, nil
		}
	}
}
