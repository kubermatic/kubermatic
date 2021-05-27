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

package mla

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func newCleanupReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	cleanupController *cleanupController,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &cleanupReconciler{
		Client:            client,
		log:               log,
		workerName:        workerName,
		recorder:          mgr.GetEventRecorderFor(ControllerName),
		versions:          versions,
		cleanupController: cleanupController,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}
	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: "identifier", Namespace: ""}}
	src := source.Func(func(ctx context.Context, h handler.EventHandler, q workqueue.RateLimitingInterface, p ...predicate.Predicate) error {
		q.Add(request)
		return nil
	})

	if err := c.Watch(src, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch: %w", err)
	}

	return nil
}

type cleanupReconciler struct {
	ctrlruntimeclient.Client
	log               *zap.SugaredLogger
	workerName        string
	recorder          record.EventRecorder
	versions          kubermatic.Versions
	cleanupController *cleanupController
}

func (r *cleanupReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	if err := r.cleanupController.cleanup(ctx); err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to cleanup: %w", err)
	}

	return reconcile.Result{}, nil
}

type cleanupController struct {
	ctrlruntimeclient.Client
	log      *zap.SugaredLogger
	cleaners []cleaner
}

func newCleanupController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	cleaners ...cleaner,
) *cleanupController {
	return &cleanupController{
		Client:   client,
		log:      log,
		cleaners: cleaners,
	}
}

func (r *cleanupController) cleanup(ctx context.Context) error {
	for _, cleaner := range r.cleaners {
		if err := cleaner.cleanUp(ctx); err != nil {
			return err
		}
	}
	return nil
}
