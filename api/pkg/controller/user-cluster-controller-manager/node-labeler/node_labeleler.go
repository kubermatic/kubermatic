package nodelabeler

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
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
		recorder: mgr.GetRecorder(controllerName),
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

	if !labelsChanged {
		log.Debug("No label changes, not updating node")
		return nil
	}

	if err := r.updateNode(node.Name, func(n *corev1.Node) {
		n.Labels = node.Labels
	}); err != nil {
		return fmt.Errorf("failed to update node: %v", err)
	}

	return nil
}

func (r *reconciler) updateNode(name string, modify func(*corev1.Node)) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		//Get latest version
		node := &corev1.Node{}
		if err := r.client.Get(r.ctx, types.NamespacedName{Name: name}, node); err != nil {
			return err
		}
		// Apply modifications
		modify(node)
		// Update the cluster
		return r.client.Update(r.ctx, node)
	})
}
