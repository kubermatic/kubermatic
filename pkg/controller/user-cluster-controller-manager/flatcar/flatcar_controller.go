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

	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/flatcar/resources"
	nodelabelerapi "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/node-labeler/api"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/util"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_flatcar_controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client
	overwriteRegistry string
	updateWindow      kubermaticv1.UpdateWindow
	clusterIsPaused   util.IsPausedChecker
}

func Add(mgr manager.Manager, overwriteRegistry string, updateWindow kubermaticv1.UpdateWindow, clusterIsPaused util.IsPausedChecker) error {

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

	if err := r.reconcileUpdateOperatorResources(ctx); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile the UpdateOperator resources: %v", err)
	}

	return reconcile.Result{}, nil
}

// reconcileUpdateOperatorResources deploys the FlatcarUpdateOperator
// https://github.com/kinvolk/flatcar-linux-update-operator
func (r *Reconciler) reconcileUpdateOperatorResources(ctx context.Context) error {
	saCreators := []reconciling.NamedServiceAccountCreatorGetter{
		resources.OperatorServiceAccountCreator(),
		resources.AgentServiceAccountCreator(),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, saCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the ServiceAccounts: %v", err)
	}

	crCreators := []reconciling.NamedClusterRoleCreatorGetter{
		resources.OperatorClusterRoleCreator(),
		resources.AgentClusterRoleCreator(),
	}
	if err := reconciling.ReconcileClusterRoles(ctx, crCreators, metav1.NamespaceNone, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the ClusterRoles: %v", err)
	}

	crbCreators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		resources.OperatorClusterRoleBindingCreator(),
		resources.AgentClusterRoleBindingCreator(),
	}
	if err := reconciling.ReconcileClusterRoleBindings(ctx, crbCreators, metav1.NamespaceNone, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the ClusterRoleBindings: %v", err)
	}

	depCreators := getDeploymentCreators(r.overwriteRegistry, r.updateWindow)
	if err := reconciling.ReconcileDeployments(ctx, depCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the Deployments: %v", err)
	}

	dsCreators := getDaemonSetCreators(r.overwriteRegistry)
	if err := reconciling.ReconcileDaemonSets(ctx, dsCreators, metav1.NamespaceSystem, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the DaemonSets: %v", err)
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

func getDeploymentCreators(overwriteRegistry string, updateWindow kubermaticv1.UpdateWindow) []reconciling.NamedDeploymentCreatorGetter {
	return []reconciling.NamedDeploymentCreatorGetter{
		resources.OperatorDeploymentCreator(getRegistryDefaultFunc(overwriteRegistry), updateWindow),
	}
}

func getDaemonSetCreators(overwriteRegistry string) []reconciling.NamedDaemonSetCreatorGetter {
	return []reconciling.NamedDaemonSetCreatorGetter{
		resources.AgentDaemonSetCreator(getRegistryDefaultFunc(overwriteRegistry)),
	}
}
