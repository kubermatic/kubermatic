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

package mlacontroller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type cleaner interface {
	Cleanup(context.Context) error
}

// cleanupReconciler is a "meta" reconciler which is used when MLA is globally disabled;
// it will re-use all the other reconcilers and call their .Cleanup() function exactly
// once during application startup (i.e. it does not permanently watch any resources).
type cleanupReconciler struct {
	seedClient ctrlruntimeclient.Client
	log        *zap.SugaredLogger
	cleaners   []cleaner
}

func newCleanupReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	cleaners ...cleaner,
) *cleanupReconciler {
	return &cleanupReconciler{
		seedClient: mgr.GetClient(),
		log:        log.Named("cleanup"),
		cleaners:   cleaners,
	}
}

func (r *cleanupReconciler) Start(ctx context.Context, mgr manager.Manager, workers int) error {
	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: workers,
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

func (r *cleanupReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.log.Debug("Processing")

	if err := r.reconcile(ctx); err != nil {
		return reconcile.Result{RequeueAfter: 1 * time.Minute}, fmt.Errorf("unable to cleanup: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *cleanupReconciler) reconcile(ctx context.Context) error {
	for _, cleaner := range r.cleaners {
		if err := cleaner.Cleanup(ctx); err != nil {
			return err
		}
	}

	return nil
}
