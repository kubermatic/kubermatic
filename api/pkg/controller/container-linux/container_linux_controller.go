package containerlinux

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/controller/container-linux/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ControllerName = "kubermatic_container_linux_controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client
	overwriteRegistry string
}

func Add(mgr manager.Manager, overwriteRegistry string) error {

	reconciler := &Reconciler{
		Client:            mgr.GetClient(),
		overwriteRegistry: overwriteRegistry,
	}

	ctrlOptions := controller.Options{
		Reconciler: reconciler,
		// Only use 1 worker to prevent concurrent operator deployments
		MaxConcurrentReconciles: 1,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	predicates := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			newNode := e.ObjectNew.(*corev1.Node)
			// Only sync if the node uses ContainerLinux & is missing the label
			return isContainerLinuxNode(newNode) && !hasContainerLinuxLabel(newNode)
		},
	}
	return c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{}, predicates)
}

func (r *Reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hasContainerLinuxNodes, err := r.labelAllContainerLinuxNodes(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure that all ContainerLinux nodes have the %s label: %v", resources.NodeSelectorLabelKey, err)
	}

	// If we have no ContainerLinux node, we just skip the rest
	if !hasContainerLinuxNodes {
		return reconcile.Result{}, nil
	}

	if err := r.reconcileUpdateOperatorResources(ctx); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile the UpdateOperator resources: %v", err)
	}

	return reconcile.Result{}, nil
}

// labelAllContainerLinuxNodes labels all nodes which use ContainerLinux.
// If at least one ContainerLinux node exists, true will be returned
func (r *Reconciler) labelAllContainerLinuxNodes(ctx context.Context) (bool, error) {
	var hasContainerLinuxNode bool
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList); err != nil {
		return false, fmt.Errorf("failed to list nodes: %v", err)
	}

	for _, node := range nodeList.Items {
		usesContainerLinux := isContainerLinuxNode(&node)
		hasLabel := hasContainerLinuxLabel(&node)

		if usesContainerLinux && !hasLabel {
			oldNode := node.DeepCopy()
			if node.Labels == nil {
				node.Labels = map[string]string{}
			}
			node.Labels[resources.NodeSelectorLabelKey] = resources.NodeSelectorLabelValue
			if err := r.Client.Patch(ctx, &node, ctrlruntimeclient.MergeFrom(oldNode)); err != nil {
				return false, fmt.Errorf("failed to update node %q: %v", node.Name, err)
			}
		}

		if usesContainerLinux {
			hasContainerLinuxNode = true
		}
	}

	return hasContainerLinuxNode, nil
}

// reconcileUpdateOperatorResources deploys the ContainerLinuxUpdateOperator
// https://github.com/coreos/container-linux-update-operator
func (r *Reconciler) reconcileUpdateOperatorResources(ctx context.Context) error {
	saCreators := []reconciling.NamedServiceAccountCreatorGetter{
		resources.ServiceAccountCreator(),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, saCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the ServiceAccounts: %v", err)
	}

	crCreators := []reconciling.NamedClusterRoleCreatorGetter{
		resources.ClusterRoleCreator(),
	}
	if err := reconciling.ReconcileClusterRoles(ctx, crCreators, metav1.NamespaceNone, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the ClusterRoles: %v", err)
	}

	crbCreators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		resources.ClusterRoleBindingCreator(),
	}
	if err := reconciling.ReconcileClusterRoleBindings(ctx, crbCreators, metav1.NamespaceNone, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the ClusterRoleBindings: %v", err)
	}

	depCreators := GetDeploymentCreators(r.overwriteRegistry)
	if err := reconciling.ReconcileDeployments(ctx, depCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the Deployments: %v", err)
	}

	dsCreators := GetDaemonSetCreators(r.overwriteRegistry)
	if err := reconciling.ReconcileDaemonSets(ctx, dsCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the DaemonSet: %v", err)
	}

	return nil
}

func getRegistryDefaultFunc(overwriteRegistry string) func(defaultRegistry string) string {
	return func(defaultRegistry string) string {
		if overwriteRegistry != "" {
			return overwriteRegistry
		}
		return defaultRegistry
	}
}

func isContainerLinuxNode(node *corev1.Node) bool {
	osImage := strings.ToLower(node.Status.NodeInfo.OSImage)
	return strings.Contains(osImage, "container linux")
}

func hasContainerLinuxLabel(node *corev1.Node) bool {
	return node.Labels[resources.NodeSelectorLabelKey] == resources.NodeSelectorLabelValue
}

func GetDeploymentCreators(overwriteRegistry string) []reconciling.NamedDeploymentCreatorGetter {
	return []reconciling.NamedDeploymentCreatorGetter{
		resources.DeploymentCreator(getRegistryDefaultFunc(overwriteRegistry)),
	}
}
func GetDaemonSetCreators(overwriteRegistry string) []reconciling.NamedDaemonSetCreatorGetter {
	return []reconciling.NamedDaemonSetCreatorGetter{
		resources.DaemonSetCreator(getRegistryDefaultFunc(overwriteRegistry)),
	}
}
