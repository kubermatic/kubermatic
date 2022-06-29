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

package projectsynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
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
	ControllerName = "kkp-project-synchronizer"
)

type reconciler struct {
	log             *zap.SugaredLogger
	recorder        record.EventRecorder
	masterClient    ctrlruntimeclient.Client
	masterAPIClient ctrlruntimeclient.Reader
	seedClients     map[string]ctrlruntimeclient.Client
}

func Add(
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	r := &reconciler{
		log:             log.Named(ControllerName),
		recorder:        masterManager.GetEventRecorderFor(ControllerName),
		masterClient:    masterManager.GetClient(),
		masterAPIClient: masterManager.GetAPIReader(),
		seedClients:     map[string]ctrlruntimeclient.Client{},
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	c, err := controller.New(ControllerName, masterManager, controller.Options{Reconciler: r, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Project{}},
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return fmt.Errorf("failed to create watch for projects: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Seed{}},
		enqueueAllProjects(r.masterClient, r.log),
	); err != nil {
		return fmt.Errorf("failed to create watch for seeds: %w", err)
	}

	return nil
}

// Reconcile reconciles Kubermatic Project objects on the master cluster to all seed clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	project := &kubermaticv1.Project{}
	if err := r.masterAPIClient.Get(ctx, request.NamespacedName, project); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if project.Status.Phase == "" {
		log.Debug("Project has no phase set in its status, skipping reconciling")
		return reconcile.Result{}, nil
	}

	if !project.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, project); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, project, apiv1.SeedProjectCleanupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	projectCreatorGetters := []reconciling.NamedKubermaticV1ProjectCreatorGetter{
		projectCreatorGetter(project),
	}

	err := r.syncAllSeeds(log, project, func(seedClusterClient ctrlruntimeclient.Client, project *kubermaticv1.Project) error {
		err := reconciling.EnsureNamedObjects(ctx, seedClusterClient, "", projectCreatorGetters)
		if err != nil {
			return fmt.Errorf("failed to reconcile project: %w", err)
		}

		seedProject := &kubermaticv1.Project{}
		if err := seedClusterClient.Get(ctx, request.NamespacedName, seedProject); err != nil {
			return fmt.Errorf("failed to fetch project on seed cluster: %w", err)
		}

		oldProject := seedProject.DeepCopy()
		seedProject.Status = project.Status
		if err := seedClusterClient.Status().Patch(ctx, seedProject, ctrlruntimeclient.MergeFrom(oldProject)); err != nil {
			return fmt.Errorf("failed to update project status on seed cluster: %w", err)
		}

		return nil
	})

	if err != nil {
		r.recorder.Eventf(project, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconciled project: %s: %w", project.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, project *kubermaticv1.Project) error {
	err := r.syncAllSeeds(log, project, func(seedClusterClient ctrlruntimeclient.Client, project *kubermaticv1.Project) error {
		return ctrlruntimeclient.IgnoreNotFound(seedClusterClient.Delete(ctx, project))
	})
	if err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, project, apiv1.SeedProjectCleanupFinalizer)
}

func (r *reconciler) syncAllSeeds(
	log *zap.SugaredLogger,
	project *kubermaticv1.Project,
	action func(seedClusterClient ctrlruntimeclient.Client, project *kubermaticv1.Project) error,
) error {
	for seedName, seedClient := range r.seedClients {
		if err := action(seedClient, project); err != nil {
			return fmt.Errorf("failed syncing project for seed %s: %w", seedName, err)
		}
		log.Debugw("Reconciled project with seed", "seed", seedName)
	}
	return nil
}

func enqueueAllProjects(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		projectList := &kubermaticv1.ProjectList{}
		if err := client.List(context.Background(), projectList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list projects: %w", err))
		}
		for _, project := range projectList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name: project.Name,
			}})
		}
		return requests
	})
}
