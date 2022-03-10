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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-project-controller"
)

type Reconciler struct {
	client.Client
	log *zap.SugaredLogger
}

func Add(mgr manager.Manager, log *zap.SugaredLogger, workerCount int) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),
		log:    log,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: workerCount})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, &handler.EnqueueRequestForObject{},
		predicate.NewPredicateFuncs(func(o client.Object) bool {
			// Skip reconciliation for projects that have a value for .status.Phase
			project := o.(*kubermaticv1.Project)
			return project.Status.Phase == ""
		})); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	log := r.log.With("request", req)

	log.Info("Reconciling Project")

	project := &kubermaticv1.Project{}
	if err := r.Get(ctx, req.NamespacedName, project); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err := r.setProjectStatus(ctx, log, project)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) setProjectStatus(ctx context.Context, log *zap.SugaredLogger, project *kubermaticv1.Project) error {
	log = log.With("project", project.Name)

	if project.Status.Phase == "" {
		project.Status.Phase = kubermaticv1.ProjectInactive
	}

	log.Debugf("Setting initial status %s", kubermaticv1.ProjectInactive)

	return r.Status().Update(ctx, project)
}
