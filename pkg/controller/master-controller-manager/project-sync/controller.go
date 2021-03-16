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

package projectsync

import (
	"context"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

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
	ControllerName = "project-sync-controller"
)

type reconciler struct {
	log                     *zap.SugaredLogger
	recorder                record.EventRecorder
	masterClient            ctrlruntimeclient.Client
	seedClientGetter        provider.SeedClientGetter
	workerNameLabelSelector labels.Selector
}

func Add(mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	seedKubeconfigGetter provider.SeedKubeconfigGetter) error {

	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %v", err)
	}

	reconciler := &reconciler{
		log:                     log.Named(ControllerName),
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		masterClient:            mgr.GetClient(),
		seedClientGetter:        provider.SeedClientGetterFactory(seedKubeconfigGetter),
		workerNameLabelSelector: workerSelector,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Project{}},
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return fmt.Errorf("failed to create watch for Projects: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Seed{}},
		enqueueAllProjects(reconciler.masterClient, reconciler.log),
	); err != nil {
		return fmt.Errorf("failed to create seed watcher: %v", err)
	}

	return nil
}

// Reconcile reconciles Kubermatic Project objects on the master cluster to all seed clusters
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	project := &kubermaticv1.Project{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, project); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !project.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, project); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %v", err)
		}
	}

	kuberneteshelper.AddFinalizer(project, kubermaticapiv1.SeedProjectCleanupFinalizer)
	if err := r.masterClient.Update(ctx, project); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add Project finalizer %s: %v", project.Name, err)
	}

	projectCreatorGetter := []reconciling.NamedKubermaticV1ProjectCreatorGetter{
		projectCreatorGetter(project),
	}

	err := r.syncAllSeeds(ctx, log, project, func(seedClusterClient ctrlruntimeclient.Client, project *kubermaticv1.Project) error {
		workerNameLabelSelectorRequirements, _ := r.workerNameLabelSelector.Requirements()
		projectLabelRequirement, err := labels.NewRequirement(kubermaticv1.ProjectIDLabelKey, selection.Equals, []string{project.Name})
		if err != nil {
			return fmt.Errorf("failed to construct label requirement for project: %v", err)
		}

		listOpts := &ctrlruntimeclient.ListOptions{
			LabelSelector: labels.NewSelector().Add(append(workerNameLabelSelectorRequirements, *projectLabelRequirement)...),
		}
		clusterList := &kubermaticv1.ClusterList{}
		if err := seedClusterClient.List(ctx, clusterList, listOpts); err != nil {
			return fmt.Errorf("failed to list Cluster objects: %v", err)
		}
		if len(clusterList.Items) == 0 {
			// No cluster within this Project is found in this Seed, so in this case, we don't sync this Project to Seed.
			return nil
		}
		return reconciling.ReconcileKubermaticV1Projects(ctx, projectCreatorGetter, "", seedClusterClient)
	})

	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconciled Project: %s: %v", project.Name, err)
	}
	return reconcile.Result{}, nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, project *kubermaticv1.Project) error {
	err := r.syncAllSeeds(ctx, log, project, func(seedClusterClient ctrlruntimeclient.Client, project *kubermaticv1.Project) error {
		if err := seedClusterClient.Delete(ctx, project); err != nil {
			return ctrlruntimeclient.IgnoreNotFound(err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	kuberneteshelper.RemoveFinalizer(project, kubermaticapiv1.SeedProjectCleanupFinalizer)
	if err := r.masterClient.Update(ctx, project); err != nil {
		return fmt.Errorf("failed to remove Project finalizer %s: %v", project.Name, err)
	}
	return nil
}

func (r *reconciler) syncAllSeeds(
	ctx context.Context,
	log *zap.SugaredLogger,
	project *kubermaticv1.Project,
	action func(seedClusterClient ctrlruntimeclient.Client, project *kubermaticv1.Project) error) error {

	seedList := &kubermaticv1.SeedList{}
	if err := r.masterClient.List(ctx, seedList); err != nil {
		return fmt.Errorf("failed listing seeds: %w", err)
	}

	for _, seed := range seedList.Items {
		seedClient, err := r.seedClientGetter(&seed)
		if err != nil {
			return fmt.Errorf("failed getting seed client for Seed %s: %w", seed.Name, err)
		}

		err = action(seedClient, project)
		if err != nil {
			return fmt.Errorf("failed syncing Project for Seed %s: %w", seed.Name, err)
		}
		log.Debugw("Reconciled Project with Seed", "seed", seed.Name)
	}
	return nil
}

func projectCreatorGetter(project *kubermaticv1.Project) reconciling.NamedKubermaticV1ProjectCreatorGetter {
	return func() (string, reconciling.KubermaticV1ProjectCreator) {
		return project.Name, func(p *kubermaticv1.Project) (*kubermaticv1.Project, error) {
			p.Name = project.Name
			p.Spec = project.Spec
			return p, nil
		}
	}
}

func enqueueAllProjects(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		projectList := &kubermaticv1.ProjectList{}
		if err := client.List(context.Background(), projectList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list Projects: %v", err))
		}
		for _, project := range projectList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name: project.Name,
			}})
		}
		return requests
	})
}
