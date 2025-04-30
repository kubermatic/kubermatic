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

package presetcontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-preset-controller"
)

type reconciler struct {
	log                     *zap.SugaredLogger
	workerNameLabelSelector labels.Selector
	workerName              string
	recorder                record.EventRecorder
	seedClient              ctrlruntimeclient.Client
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &reconciler{
		log:                     log.Named(ControllerName),
		workerNameLabelSelector: workerSelector,
		workerName:              workerName,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		seedClient:              mgr.GetClient(),
	}

	_, err = builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Preset{}).
		Build(reconciler)

	return err
}

// Reconcile reconciles the kubermatic cluster template instance in the seed cluster.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	preset := &kubermaticv1.Preset{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, preset); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get preset %s: %w", request.NamespacedName, err)
	}

	err := r.reconcile(ctx, preset, log)
	if err != nil {
		r.recorder.Event(preset, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, preset *kubermaticv1.Preset, log *zap.SugaredLogger) error {
	// handle deletion to change all cluster annotation
	if !preset.DeletionTimestamp.IsZero() {
		log.Debug("The preset was deleted")
		workerNameLabelSelectorRequirements, _ := r.workerNameLabelSelector.Requirements()
		presetLabelRequirement, err := labels.NewRequirement(kubermaticv1.IsCredentialPresetLabelKey, selection.Equals, []string{"true"})
		if err != nil {
			return fmt.Errorf("failed to construct label requirement for credential preset: %w", err)
		}

		listOpts := &ctrlruntimeclient.ListOptions{
			LabelSelector: labels.NewSelector().Add(append(workerNameLabelSelectorRequirements, *presetLabelRequirement)...),
		}

		clusters := &kubermaticv1.ClusterList{}
		if err := r.seedClient.List(ctx, clusters, listOpts); err != nil {
			return fmt.Errorf("failed to get clusters %w", err)
		}
		log.Debug("Update clusters after preset deletion")
		for _, cluster := range clusters.Items {
			if cluster.Annotations != nil && cluster.Annotations[kubermaticv1.PresetNameAnnotation] == preset.Name {
				log.Debugw("Update cluster", "cluster", cluster.Name)
				copyCluster := cluster.DeepCopy()
				copyCluster.Annotations[kubermaticv1.PresetInvalidatedAnnotation] = string(kubermaticv1.PresetDeleted)
				if err := r.seedClient.Update(ctx, copyCluster); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
