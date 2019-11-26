package containerlinux

import (
	"context"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/container-linux/resources"
	nodelabelerapi "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/node-labeler/api"
	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

	predicates := predicateutil.Factory(func(m metav1.Object, _ runtime.Object) bool {
		return m.GetLabels()[nodelabelerapi.DistributionLabelKey] == nodelabelerapi.ContainerLinuxLabelValue
	})
	return c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{}, predicates)
}

func (r *Reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := r.reconcileUpdateOperatorResources(ctx); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile the UpdateOperator resources: %v", err)
	}

	return reconcile.Result{}, nil
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
