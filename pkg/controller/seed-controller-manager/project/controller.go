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

package project

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-project-controller"

	// CleanupFinalizer is put on Projects to ensure all clusters
	// are deleted before a Project can be deleted.
	CleanupFinalizer = "kubermatic.k8c.io/cleanup-clusters"
)

type Reconciler struct {
	ctrlruntimeclient.Client
	log      *zap.SugaredLogger
	recorder events.EventRecorder
}

func Add(mgr manager.Manager, log *zap.SugaredLogger, workerCount int) error {
	reconciler := &Reconciler{
		Client:   mgr.GetClient(),
		log:      log,
		recorder: mgr.GetEventRecorder(ControllerName),
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: workerCount,
		}).
		For(&kubermaticv1.Project{}).
		Watches(&kubermaticv1.Cluster{}, util.EnqueueProjectForCluster()).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	log := r.log.With("project", req.Name)
	log.Debug("Reconciling")

	project := &kubermaticv1.Project{}
	if err := r.Get(ctx, req.NamespacedName, project); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err := r.reconcile(ctx, log, project)
	if err != nil {
		r.recorder.Eventf(project, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, project *kubermaticv1.Project) error {
	if project.DeletionTimestamp != nil {
		return r.handleCleanup(ctx, log, project)
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r, project, CleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	return nil
}

func (r *Reconciler) handleCleanup(ctx context.Context, log *zap.SugaredLogger, project *kubermaticv1.Project) error {
	log.Debug("Handling project deletion")

	// delete all clusters in this project
	clusters := &kubermaticv1.ClusterList{}
	selector := labels.SelectorFromSet(map[string]string{kubermaticv1.ProjectIDLabelKey: project.Name})
	listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: selector}
	if err := r.List(ctx, clusters, listOpts); err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, cluster := range clusters.Items {
		if cluster.DeletionTimestamp == nil {
			if err := r.Delete(ctx, &cluster); err != nil {
				return fmt.Errorf("failed to delete cluster %s: %w", cluster.Name, err)
			}
		}
	}

	// we're done!
	if len(clusters.Items) == 0 {
		log.Info("All clusters in project have been deleted.")

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r, project, CleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove finalizer: %w", err)
		}
	}

	// there are still clusters remaining;
	// since we watch Cluster objects, we get triggered when they are deleted
	return nil
}
