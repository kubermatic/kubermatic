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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller syncs the kubermatic constraint templates on the master cluster to the seed clusters.
	ControllerName = "kkp-master-constraint-template-controller"

	// cleanupFinalizer indicates that synced gatekeeper Constraint Templates on seed clusters need cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-gatekeeper-master-constraint-templates"
)

type reconciler struct {
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	masterClient ctrlruntimeclient.Client
	namespace    string
	seedClients  kuberneteshelper.SeedClientMap
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	namespace string,
	seedManagers map[string]manager.Manager,
) error {
	reconciler := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     mgr.GetEventRecorderFor(ControllerName),
		masterClient: mgr.GetClient(),
		namespace:    namespace,
		seedClients:  kuberneteshelper.SeedClientMap{},
	}

	for seedName, seedManager := range seedManagers {
		reconciler.seedClients[seedName] = seedManager.GetClient()
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.ConstraintTemplate{}).
		Watches(&kubermaticv1.Seed{}, enqueueAllConstraintTemplates(reconciler.masterClient, reconciler.log), builder.WithPredicates(predicate.ByNamespace(namespace))).
		Build(reconciler)

	return err
}

// Reconcile reconciles the kubermatic constraint template on the master cluster to all seed clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("template", request.Name)
	log.Debug("Reconciling")

	constraintTemplate := &kubermaticv1.ConstraintTemplate{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, constraintTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("constraint template not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get constraint template %s: %w", constraintTemplate.Name, err)
	}

	err := r.reconcile(ctx, log, constraintTemplate)
	if err != nil {
		r.recorder.Event(constraintTemplate, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, constraintTemplate *kubermaticv1.ConstraintTemplate) error {
	if constraintTemplate.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(constraintTemplate, cleanupFinalizer) {
			return nil
		}

		err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
			err := seedClient.Delete(ctx, &kubermaticv1.ConstraintTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: constraintTemplate.Name,
				},
			})

			return ctrlruntimeclient.IgnoreNotFound(err)
		})
		if err != nil {
			return err
		}

		return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, constraintTemplate, cleanupFinalizer)
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, constraintTemplate, cleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	ctReconcilerFactories := []reconciling.NamedConstraintTemplateReconcilerFactory{
		constraintTemplateReconcilerFactory(constraintTemplate),
	}

	return r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		seedCT := &kubermaticv1.ConstraintTemplate{}
		if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(constraintTemplate), seedCT); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch ConstraintTemplate on seed cluster: %w", err)
		}

		// see project-synchronizer's syncAllSeeds comment
		if seedCT.UID != "" && seedCT.UID == constraintTemplate.UID {
			return nil
		}

		return reconciling.ReconcileConstraintTemplates(ctx, ctReconcilerFactories, "", seedClient)
	})
}

func constraintTemplateReconcilerFactory(kubeCT *kubermaticv1.ConstraintTemplate) reconciling.NamedConstraintTemplateReconcilerFactory {
	return func() (string, reconciling.ConstraintTemplateReconciler) {
		return kubeCT.Name, func(ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error) {
			ct.Name = kubeCT.Name
			ct.Spec = kubeCT.Spec

			return ct, nil
		}
	}
}

func enqueueAllConstraintTemplates(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		ctList := &kubermaticv1.ConstraintTemplateList{}
		if err := client.List(ctx, ctList); err != nil {
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
