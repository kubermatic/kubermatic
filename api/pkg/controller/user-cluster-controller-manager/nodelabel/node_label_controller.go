package nodelabel

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/nodelabel/resources"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
	// ControllerName is the name of the controller.
	ControllerName = "kubermatic_node_label_controller"
)

// Reconciler is the node label
type Reconciler struct {
	log *zap.SugaredLogger
	ctrlruntimeclient.Client
}

// Add adds the reconciler to the controller.
func Add(log *zap.SugaredLogger, mgr manager.Manager) error {
	reconciler := &Reconciler{
		log:    log,
		Client: mgr.GetClient(),
	}

	ctrlOpts := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: 1,
	}

	c, err := controller.New(ControllerName, mgr, ctrlOpts)
	if err != nil {
		return err
	}

	predicates := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// TODO: Add proper func when they exist.
			return true
		},
	}

	return c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{}, predicates)
}

// Reconcile is the main reconcilliation loop for the controller.
func (r *Reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := r.labelAllNodes(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure all nodes have the %s label: %v", resources.DistributionLabelKey, err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) labelAllNodes(ctx context.Context) error {
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList); err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	for _, node := range nodeList.Items {
		label, err := detectLabel(&node)
		if err != nil {
			return fmt.Errorf("could not detect label on node %q: %v", node.Name, err)
		}
		hasLabel := hasCorrectDistributionLabel(&node, label)

		if !hasLabel {
			err := r.applyLabel(ctx, &node, label)
			if err != nil {
				return fmt.Errorf("could not apply label on node %q: %v", node.Name, err)
			}
		}
	}

	return nil
}

func (r *Reconciler) applyLabel(ctx context.Context, node *corev1.Node, label string) error {
	oldNode := node.DeepCopy()
	if node.Labels == nil {
		node.Labels = map[string]string{}
	}
	if node.Labels[resources.DistributionLabelKey] != label {
		r.log.Debugf("Changing label %q from value %q to value %q", resources.DistributionLabelKey, node.Labels[resources.DistributionLabelKey], label)
		node.Labels[resources.DistributionLabelKey] = label
		if err := r.Client.Patch(ctx, node, ctrlruntimeclient.MergeFrom(oldNode)); err != nil {
			return fmt.Errorf("failed to update node: %v", err)
		}
	}

	r.log.Debug("No label changes, not updating node.")

	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		newNode := &corev1.Node{}
		if err := r.Get(ctx, types.NamespacedName{Name: node.Name}, newNode); err != nil {
			return err
		}

		return r.Client.Update(ctx, newNode)
	}); err != nil {
		return fmt.Errorf("failed to update node %q: %v", node.Name, err)
	}

	return nil
}

func detectLabel(node *corev1.Node) (string, error) {
	osImage := strings.ToLower(node.Status.NodeInfo.OSImage)
	for k, v := range resources.OSLabelMatchValues {
		if strings.Contains(osImage, v) {
			return k, nil
		}
	}
	return "", fmt.Errorf("Could not detect distribution from image name %s", node.Status.NodeInfo.OSImage)
}

func hasCorrectDistributionLabel(node *corev1.Node, label string) bool {
	return node.Labels[resources.DistributionLabelKey] == label
}
