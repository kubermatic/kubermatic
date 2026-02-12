/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package nodelabeler

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/node-labeler/api"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller creates events on the nodes, so do not put the word Kubermatic in it.
	controllerName = "kkp-node-labeler"
)

type reconciler struct {
	log      *zap.SugaredLogger
	client   ctrlruntimeclient.Client
	recorder events.EventRecorder
	labels   map[string]string
}

func Add(ctx context.Context, log *zap.SugaredLogger, mgr manager.Manager, labels map[string]string) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:      log,
		client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorder(controllerName),
		labels:   labels,
	}

	// Ignore update events that don't touch the labels
	labelsChangedPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return !apiequality.Semantic.DeepEqual(e.ObjectOld.GetLabels(), e.ObjectNew.GetLabels())
		},
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName).
		For(&corev1.Node{}, builder.WithPredicates(labelsChangedPredicate)).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("Node", request.Name)
	log.Debug("Reconciling")

	node := &corev1.Node{}
	if err := r.client.Get(ctx, request.NamespacedName, node); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Node not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get node: %w", err)
	}

	err := r.reconcile(ctx, log, node)
	if err != nil {
		r.recorder.Eventf(node, nil, corev1.EventTypeWarning, "ApplyingLabelsFailed", "Reconciling", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, node *corev1.Node) error {
	oldNode := node.DeepCopy()
	var labelsChanged bool
	if node.Labels == nil {
		node.Labels = map[string]string{}
	}
	for key, value := range r.labels {
		if node.Labels[key] != value {
			log.Debugw("Setting label", "label-key", key, "old-label-value", node.Labels[key], "new-label-value", value)
			labelsChanged = true
			node.Labels[key] = value
		}
	}

	distributionLabelChanged, err := applyDistributionLabel(log, node)
	if err != nil {
		return fmt.Errorf("failed to apply distribution label: %w", err)
	}

	labelsChanged = labelsChanged || distributionLabelChanged
	if !labelsChanged {
		log.Debug("No label changes, not updating node")
		return nil
	}

	if err := r.client.Patch(ctx, node, ctrlruntimeclient.MergeFrom(oldNode)); err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	return nil
}

func applyDistributionLabel(log *zap.SugaredLogger, node *corev1.Node) (changed bool, err error) {
	osImage := node.Status.NodeInfo.OSImage

	wantedLabel := findDistributionLabel(node.Status.NodeInfo.OSImage)
	if wantedLabel == "" {
		return false, fmt.Errorf("could not detect distribution from image name %q", osImage)
	}

	if node.Labels[api.DistributionLabelKey] == wantedLabel {
		return false, nil
	}

	node.Labels[api.DistributionLabelKey] = wantedLabel
	log.Debugw("Setting label", "label-key", api.DistributionLabelKey, "old-label-value", node.Labels[api.DistributionLabelKey], "new-label-value", wantedLabel)
	return true, nil
}

// findDistributionLabel finds the best label value for a given
// OS image string. It returns the longest match to ensure consistent
// results.
func findDistributionLabel(osImage string) string {
	osImage = strings.ToLower(osImage)

	matchedLabel := ""
	matchedValue := ""
	for k, v := range api.OSLabelMatchValues {
		for _, substring := range v {
			if strings.Contains(osImage, substring) && len(substring) > len(matchedValue) {
				matchedLabel = k
				matchedValue = substring
			}
		}
	}

	return matchedLabel
}
