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

package kcstatuscontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this controller.
	ControllerName = "kkp-kc-status-controller"
)

// Reconciler watches the Kubermatic Configuration status and updates the Kubermatic Edition and Kubermatic Version.
type Reconciler struct {
	ctrlruntimeclient.Client
	log      *zap.SugaredLogger
	recorder record.EventRecorder
	versions kubermatic.Versions
}

// Add creates a new Kubermatic Configuration status controller and sets up watches.
func Add(
	ctx context.Context,
	mgr manager.Manager,
	numWorkers int,
	log *zap.SugaredLogger,
	namespace string,
	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		Client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorderFor(ControllerName),
		log:      log.Named(ControllerName),
		versions: versions,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	nsPredicate := predicate.ByNamespace(namespace)

	// watch the Kubermatic Configuration in the given namespace
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.KubermaticConfiguration{}},
		&handler.EnqueueRequestForObject{},
		nsPredicate,
	); err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := r.log.With("configuration", request.Name)
	logger.Debug("Reconciling")

	kc := &kubermaticv1.KubermaticConfiguration{}

	if err := r.Get(ctx, request.NamespacedName, kc); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get kubermatic configuration: %w", err)
	}

	if kc.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	if err := r.reconcile(ctx, logger, kc); err != nil {
		r.recorder.Event(kc, corev1.EventTypeWarning, "ReconcilingFailed", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcile kubermatic configuration %s: %w", kc.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, kc *kubermaticv1.KubermaticConfiguration) error {
	return kubermaticv1helper.UpdateKubermaticConfigurationStatus(ctx, r, kc, func(config *kubermaticv1.KubermaticConfiguration) {
		config.Status.KubermaticEdition = r.versions.KubermaticEdition.ShortString()
		config.Status.KubermaticVersion = r.versions.KubermaticCommit
	})
}
