package nodelabeler

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/node-labeler/api"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller creates events on the nodes, so do not put the word Kubermatic in it
	controllerName = "node_label_controller"
)

type reconciler struct {
	ctx      context.Context
	log      *zap.SugaredLogger
	client   ctrlruntimeclient.Client
	recorder record.EventRecorder
	labels   map[string]string
}

func Add(ctx context.Context, log *zap.SugaredLogger, mgr manager.Manager, labels map[string]string) error {
	log = log.Named(controllerName)
	if len(labels) == 0 {
		log.Info("Not starting controller as there are no labels configured on the cluster")
		return nil
	}

	r := &reconciler{
		ctx:      ctx,
		log:      log,
		client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorderFor(controllerName),
		labels:   labels,
	}
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	// Ignore update events that don't touch the labels
	labelsChangedPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return !apiequality.Semantic.DeepEqual(e.MetaOld.GetLabels(), e.MetaNew.GetLabels())
		},
	}

	if err := c.Watch(
		&source.Kind{Type: &corev1.Node{}},
		&handler.EnqueueRequestForObject{},
		labelsChangedPredicate,
	); err != nil {
		return fmt.Errorf("failed to establish watch for nodes: %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("Node", request.Name)
	log.Debug("Reconciling")

	node := &corev1.Node{}
	if err := r.client.Get(r.ctx, request.NamespacedName, node); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Node not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get node: %v", err)
	}

	err := r.reconcile(log, node)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(node, corev1.EventTypeWarning, "ApplyingLabelsFailed", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(log *zap.SugaredLogger, node *corev1.Node) error {
	oldNode := node.DeepCopy()
	var labelsChanged bool
	if node.Labels == nil {
		node.Labels = map[string]string{}
	}
	for key, value := range r.labels {
		if node.Labels[key] != value {
			log.Debugf("Setting label %q from value %q to value %q", key, node.Labels[key], value)
			labelsChanged = true
			node.Labels[key] = value
		}
	}

	distributionLabelChanged, err := applyDistributionLabel(log, node)
	if err != nil {
		return fmt.Errorf("failed to apply distribution label: %v", err)
	}

	labelsChanged = labelsChanged || distributionLabelChanged
	if !labelsChanged {
		log.Debug("No label changes, not updating node")
		return nil
	}

	if err := r.client.Patch(r.ctx, node, ctrlruntimeclient.MergeFrom(oldNode)); err != nil {
		return fmt.Errorf("failed to update node: %v", err)
	}

	return nil
}

func applyDistributionLabel(log *zap.SugaredLogger, node *corev1.Node) (changed bool, err error) {
	osImage := strings.ToLower(node.Status.NodeInfo.OSImage)

	var wantValue string
	for k, v := range api.OSLabelMatchValues {
		if strings.Contains(osImage, v) {
			wantValue = k
		}
	}
	if wantValue == "" {
		return false, fmt.Errorf("Could not detect distribution from image name %q", osImage)
	}

	if node.Labels[api.DistributionLabelKey] == wantValue {
		return false, nil
	}

	node.Labels[api.DistributionLabelKey] = wantValue
	log.Debugf("Setting label %q from value %q to value %q", api.DistributionLabelKey, node.Labels[api.DistributionLabelKey], wantValue)
	return true, nil
}
