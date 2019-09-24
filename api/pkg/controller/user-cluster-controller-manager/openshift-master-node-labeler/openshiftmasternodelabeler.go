package openshiftmasternodelabeler

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "kubermatic_openshift_master_node_labeler"
	// Keep this as low as possible. The Service controller doesn't allow
	// using nodes that have the master label as backend:
	// https://github.com/kubernetes/kubernetes/issues/65618
	minMasterNodes = 1
)

type reconciler struct {
	ctx    context.Context
	log    *zap.SugaredLogger
	client ctrlruntimeclient.Client
}

func Add(ctx context.Context, log *zap.SugaredLogger, mgr manager.Manager) error {
	r := &reconciler{
		ctx:    ctx,
		log:    log.Named(controllerName),
		client: mgr.GetClient(),
	}
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
		// This controller is not safe to run with more than one worker
		// as it looks at the state of the whole cluster, which may result
		// in race bahaviour if there are more.
		MaxConcurrentReconciles: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	// Ignore update events that don't touch the metadata
	metadataChangedPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return !apiequality.Semantic.DeepEqual(e.MetaOld, e.MetaNew)
		},
	}
	if err := c.Watch(
		&source.Kind{Type: &corev1.Node{}},
		controllerutil.EnqueueConst(""),
		metadataChangedPredicate,
	); err != nil {
		return fmt.Errorf("failed to establish watch for nodes: %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	r.log.Info("Reconciling")
	result, err := r.reconcile()
	if err != nil {
		r.log.Errorw("Failed to reconcile", zap.Error(err))
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *reconciler) reconcile() (*reconcile.Result, error) {

	nodes := &corev1.NodeList{}
	if err := r.client.List(r.ctx, &ctrlruntimeclient.ListOptions{}, nodes); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %v", err)
	}

	numNodesWithMasterLabel := 0
	nodesWithoutMasterLabel := sets.String{}
	for _, node := range nodes.Items {
		if _, exists := node.Labels["node-role.kubernetes.io/master"]; exists {
			numNodesWithMasterLabel++
		} else {
			nodesWithoutMasterLabel.Insert(node.Name)
		}
	}

	if numNodesWithMasterLabel >= minMasterNodes {
		return nil, nil
	}

	for i := 0; i < minMasterNodes-numNodesWithMasterLabel; i++ {
		nodeToLabel, hasNode := nodesWithoutMasterLabel.PopAny()
		// Try again later
		if !hasNode {
			return &reconcile.Result{RequeueAfter: time.Minute}, nil
		}

		if err := r.updateNode(nodeToLabel, func(n *corev1.Node) {
			if n.Labels == nil {
				n.Labels = map[string]string{}
			}
			n.Labels["node-role.kubernetes.io/master"] = ""
		}); err != nil {
			return nil, fmt.Errorf("failed to update node %q: %v", nodeToLabel, err)
		}
	}

	return nil, nil
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
