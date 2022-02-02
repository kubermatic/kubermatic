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

package masterconstrainttemplatecontroller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// This controller syncs the kubermatic constraint templates on the master cluster to the seed clusters.
	ControllerName = "master_constraint_template_controller"
)

type reconciler struct {
	log              *zap.SugaredLogger
	recorder         record.EventRecorder
	masterClient     ctrlruntimeclient.Client
	namespace        string
	seedClientGetter provider.SeedClientGetter
}

func Add(ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	namespace string,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
) error {
	reconciler := &reconciler{
		log:              log.Named(ControllerName),
		recorder:         mgr.GetEventRecorderFor(ControllerName),
		masterClient:     mgr.GetClient(),
		namespace:        namespace,
		seedClientGetter: provider.SeedClientGetterFactory(seedKubeconfigGetter),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.ConstraintTemplate{}},
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return fmt.Errorf("failed to create watch for constraintTemplates: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Seed{}},
		enqueueAllConstraintTemplates(reconciler.masterClient, reconciler.log),
		predicate.ByNamespace(namespace),
	); err != nil {
		return fmt.Errorf("failed to create seed watcher: %w", err)
	}

	return nil
}

// Reconcile reconciles the kubermatic constraint template on the master cluster to all seed clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	constraintTemplate := &kubermaticv1.ConstraintTemplate{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, constraintTemplate); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("constraint template not found, returning")
			return reconcile.Result{}, nil
		}
		if controllerutil.IsCacheNotStarted(err) {
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get constraint template %s: %w", constraintTemplate.Name, err)
	}

	err := r.reconcile(ctx, log, constraintTemplate)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(constraintTemplate, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, constraintTemplate *kubermaticv1.ConstraintTemplate) error {
	if constraintTemplate.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperSeedConstraintTemplateCleanupFinalizer) {
			return nil
		}

		err := r.syncAllSeeds(ctx, log, constraintTemplate, func(seedClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.ConstraintTemplate) error {
			err := seedClusterClient.Delete(ctx, &kubermaticv1.ConstraintTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: constraintTemplate.Name,
				},
			})

			if kerrors.IsNotFound(err) {
				log.Debug("constraint template not found, returning")
				return nil
			}

			return err
		})
		if err != nil {
			return err
		}

		oldConstraintTemplate := constraintTemplate.DeepCopy()
		kuberneteshelper.RemoveFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperSeedConstraintTemplateCleanupFinalizer)
		if err := r.masterClient.Patch(ctx, constraintTemplate, ctrlruntimeclient.MergeFrom(oldConstraintTemplate)); err != nil {
			return fmt.Errorf("failed to remove constraint template finalizer %s: %w", constraintTemplate.Name, err)
		}
		return nil
	}

	if !kuberneteshelper.HasFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperSeedConstraintTemplateCleanupFinalizer) {
		oldConstraintTemplate := constraintTemplate.DeepCopy()
		kuberneteshelper.AddFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperSeedConstraintTemplateCleanupFinalizer)
		if err := r.masterClient.Patch(ctx, constraintTemplate, ctrlruntimeclient.MergeFrom(oldConstraintTemplate)); err != nil {
			return fmt.Errorf("failed to set constraint template finalizer %s: %w", constraintTemplate.Name, err)
		}
	}

	ctCreatorGetters := []reconciling.NamedKubermaticV1ConstraintTemplateCreatorGetter{
		constraintTemplateCreatorGetter(constraintTemplate),
	}

	return r.syncAllSeeds(ctx, log, constraintTemplate, func(seedClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.ConstraintTemplate) error {
		return reconciling.ReconcileKubermaticV1ConstraintTemplates(ctx, ctCreatorGetters, "", seedClusterClient)
	})
}

func (r *reconciler) syncAllSeeds(
	ctx context.Context,
	log *zap.SugaredLogger,
	constraintTemplate *kubermaticv1.ConstraintTemplate,
	action func(seedClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.ConstraintTemplate) error,
) error {
	seedList := &kubermaticv1.SeedList{}
	if err := r.masterClient.List(ctx, seedList, &ctrlruntimeclient.ListOptions{Namespace: r.namespace}); err != nil {
		return fmt.Errorf("failed listing seeds: %w", err)
	}

	for _, seed := range seedList.Items {
		seedClient, err := r.seedClientGetter(&seed)
		if err != nil {
			return fmt.Errorf("failed getting seed client for seed %s: %w", seed.Name, err)
		}

		err = action(seedClient, constraintTemplate)
		if err != nil {
			return fmt.Errorf("failed syncing constraint template for seed %s: %w", seed.Name, err)
		}
		log.Debugw("Reconciled constraint template with seed", "seed", seed.Name)
	}

	return nil
}

func constraintTemplateCreatorGetter(kubeCT *kubermaticv1.ConstraintTemplate) reconciling.NamedKubermaticV1ConstraintTemplateCreatorGetter {
	return func() (string, reconciling.KubermaticV1ConstraintTemplateCreator) {
		return kubeCT.Name, func(ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error) {
			ct.Name = kubeCT.Name
			ct.Spec = kubeCT.Spec

			return ct, nil
		}
	}
}

func enqueueAllConstraintTemplates(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		ctList := &kubermaticv1.ConstraintTemplateList{}
		if err := client.List(context.Background(), ctList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list constraint templates: %w", err))
		}
		for _, ct := range ctList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name: ct.Name,
			}})
		}
		return requests
	})
}
