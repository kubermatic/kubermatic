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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// ControllerName is the name of this controller.
	ControllerName = "kkp-kc-status-controller"
)

// Reconciler watches the Kubermatic Configuration status and updates the Kubermatic Edition and Kubermatic Version.
type Reconciler struct {
	ctrlruntimeclient.Client
	log      *zap.SugaredLogger
	recorder events.EventRecorder
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
		recorder: mgr.GetEventRecorder(ControllerName),
		log:      log.Named(ControllerName),
		versions: versions,
	}

	nsPredicate := predicate.ByNamespace(namespace)

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.KubermaticConfiguration{}, builder.WithPredicates(nsPredicate)).
		Build(reconciler)

	return err
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

	err := r.reconcile(ctx, logger, kc)
	if err != nil {
		r.recorder.Eventf(kc, nil, corev1.EventTypeWarning, "ReconcilingFailed", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, _ *zap.SugaredLogger, kc *kubermaticv1.KubermaticConfiguration) error {
	return util.UpdateKubermaticConfigurationStatus(ctx, r, kc, func(config *kubermaticv1.KubermaticConfiguration) {
		config.Status.KubermaticEdition = r.versions.KubermaticEdition.ShortString()
		config.Status.KubermaticVersion = r.versions.GitVersion
	})
}
