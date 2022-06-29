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

package flatcar

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/flatcar/resources"
	nodelabelerapi "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/node-labeler/api"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller is responsible for ensuring that the flatcar-linux-update-operator is installed when we have a healthy(running) flatcar
	// node in our cluster.
	ControllerName = "kkp-flatcar-update-operator-controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client
	overwriteRegistry string
	updateWindow      kubermaticv1.UpdateWindow
	clusterIsPaused   userclustercontrollermanager.IsPausedChecker
}

func Add(mgr manager.Manager, overwriteRegistry string, updateWindow kubermaticv1.UpdateWindow, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	reconciler := &Reconciler{
		Client:            mgr.GetClient(),
		overwriteRegistry: overwriteRegistry,
		updateWindow:      updateWindow,
		clusterIsPaused:   clusterIsPaused,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	predicate := predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		return o.GetLabels()[nodelabelerapi.DistributionLabelKey] == nodelabelerapi.FlatcarLabelValue
	})

	return c.Watch(&source.Kind{Type: &corev1.Node{}}, controllerutil.EnqueueConst(""), predicate)
}

func (r *Reconciler) Reconcile(ctx context.Context, _ reconcile.Request) (reconcile.Result, error) {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	var nodeList corev1.NodeList
	if err := r.List(ctx, &nodeList,
		ctrlruntimeclient.MatchingLabels{nodelabelerapi.DistributionLabelKey: nodelabelerapi.FlatcarLabelValue},
	); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to list nodes: %w", err)
	}

	// filter out any Flatcar nodes that are already being deleted
	var nodes []corev1.Node
	for _, node := range nodeList.Items {
		if node.ObjectMeta.DeletionTimestamp == nil {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) == 0 {
		if err := r.cleanupUpdateOperatorResources(ctx); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to clean up UpdateOperator resources: %w", err)
		}
	} else {
		if err := r.reconcileUpdateOperatorResources(ctx); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to reconcile the UpdateOperator resources: %w", err)
		}
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) cleanupUpdateOperatorResources(ctx context.Context) error {
	return resources.EnsureAllDeleted(ctx, r.Client)
}

// reconcileUpdateOperatorResources deploys the FlatcarUpdateOperator
// https://github.com/kinvolk/flatcar-linux-update-operator
func (r *Reconciler) reconcileUpdateOperatorResources(ctx context.Context) error {
	saCreators := []reconciling.NamedServiceAccountCreatorGetter{
		resources.OperatorServiceAccountCreator(),
		resources.AgentServiceAccountCreator(),
	}
	if err := reconciling.EnsureNamedObjects(ctx, r, metav1.NamespaceSystem, saCreators); err != nil {
		return fmt.Errorf("failed to reconcile the ServiceAccounts: %w", err)
	}

	crCreators := []reconciling.NamedClusterRoleCreatorGetter{
		resources.OperatorClusterRoleCreator(),
		resources.AgentClusterRoleCreator(),
	}
	if err := reconciling.EnsureNamedObjects(ctx, r, metav1.NamespaceNone, crCreators); err != nil {
		return fmt.Errorf("failed to reconcile the ClusterRoles: %w", err)
	}

	crbCreators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		resources.OperatorClusterRoleBindingCreator(),
		resources.AgentClusterRoleBindingCreator(),
	}
	if err := reconciling.EnsureNamedObjects(ctx, r, metav1.NamespaceNone, crbCreators); err != nil {
		return fmt.Errorf("failed to reconcile the ClusterRoleBindings: %w", err)
	}

	depCreators := getDeploymentCreators(r.overwriteRegistry, r.updateWindow)
	if err := reconciling.EnsureNamedObjects(ctx, r, metav1.NamespaceSystem, depCreators); err != nil {
		return fmt.Errorf("failed to reconcile the Deployments: %w", err)
	}

	dsCreators := getDaemonSetCreators(r.overwriteRegistry)
	if err := reconciling.EnsureNamedObjects(ctx, r, metav1.NamespaceSystem, dsCreators); err != nil {
		return fmt.Errorf("failed to reconcile the DaemonSets: %w", err)
	}

	return nil
}

func getDeploymentCreators(overwriteRegistry string, updateWindow kubermaticv1.UpdateWindow) []reconciling.NamedDeploymentCreatorGetter {
	return []reconciling.NamedDeploymentCreatorGetter{
		resources.OperatorDeploymentCreator(registry.GetOverwriteFunc(overwriteRegistry), updateWindow),
	}
}

func getDaemonSetCreators(overwriteRegistry string) []reconciling.NamedDaemonSetCreatorGetter {
	return []reconciling.NamedDaemonSetCreatorGetter{
		resources.AgentDaemonSetCreator(registry.GetOverwriteFunc(overwriteRegistry)),
	}
}
