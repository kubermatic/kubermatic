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
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	subname := "cleanup"

	reconciler := &cleanupReconciler{
		Client:            client,
		log:               log.Named(subname),
		workerName:        workerName,
		recorder:          mgr.GetEventRecorder(controllerName(subname)),
		versions:          versions,
		cleanupController: cleanupController,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName(subname)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		// trigger the controller once on startup
		WatchesRawSource(source.Func(func(ctx context.Context, rli workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
			rli.Add(reconcile.Request{NamespacedName: types.NamespacedName{Name: "identifier", Namespace: ""}})
			return nil
		})).
		Build(reconciler)

	return err
}

type cleanupReconciler struct {
	ctrlruntimeclient.Client
	log               *zap.SugaredLogger
	workerName        string
	recorder          events.EventRecorder
	versions          kubermatic.Versions
	cleanupController *cleanupController
}

func (r *cleanupReconciler) Reconcile(ctx context.Context, _ reconcile.Request) (reconcile.Result, error) {
	r.log.Debug("Processing")

	if err := r.cleanupController.Cleanup(ctx); err != nil {
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

func (r *cleanupController) Cleanup(ctx context.Context) error {
	for _, cleaner := range r.cleaners {
		if err := cleaner.CleanUp(ctx); err != nil {
			return err
		}
	}
	return nil
}
