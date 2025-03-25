//go:build e2e

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

package test

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Deploy sets up the nodeport proxy in the chosen namespace. This function
// is only used for the e2e test in pkg/test/e2e/nodeport-proxy, but kept
// here because the exact details of setting up the proxy are defined in this
// package.
// Deploy will create the necessary resources and wait for all Deployments to
// be ready.
func Deploy(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	namespace string,
	cfg *kubermaticv1.KubermaticConfiguration,
	seed *kubermaticv1.Seed,
	versions kubermatic.Versions,
	timeout time.Duration,
) error {
	if err := reconciling.ReconcileClusterRoles(ctx, []reconciling.NamedClusterRoleReconcilerFactory{
		nodeportproxy.ClusterRoleReconciler(cfg),
	}, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingReconcilerFactory{
		nodeportproxy.ClusterRoleBindingReconciler(cfg),
	}, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, []reconciling.NamedServiceAccountReconcilerFactory{
		nodeportproxy.ServiceAccountReconciler(cfg),
	}, namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAcconts: %w", err)
	}

	if err := reconciling.ReconcileRoles(ctx, []reconciling.NamedRoleReconcilerFactory{
		nodeportproxy.RoleReconciler(),
	}, namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile Roles: %w", err)
	}

	if err := reconciling.ReconcileRoleBindings(ctx, []reconciling.NamedRoleBindingReconcilerFactory{
		nodeportproxy.RoleBindingReconciler(cfg),
	}, namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBinding:s %w", err)
	}

	if err := reconciling.ReconcileServices(ctx, []reconciling.NamedServiceReconcilerFactory{
		nodeportproxy.ServiceReconciler(seed)},
		namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile Services: %w", err)
	}

	if err := reconciling.ReconcileDeployments(ctx, []reconciling.NamedDeploymentReconcilerFactory{
		nodeportproxy.EnvoyDeploymentReconciler(cfg, seed, false, versions),
		nodeportproxy.UpdaterDeploymentReconciler(cfg, seed, versions),
	}, namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %w", err)
	}

	deployments := []string{nodeportproxy.EnvoyDeploymentName, nodeportproxy.UpdaterDeploymentName}

	return wait.PollLog(ctx, log, 10*time.Second, timeout, func(ctx context.Context) (transient error, terminal error) {
		for _, name := range deployments {
			health, err := resources.HealthyDeployment(ctx, client, types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}, -1)
			if err != nil {
				return fmt.Errorf("failed to check health of %s: %w", name, err), nil
			}

			if health != kubermaticv1.HealthStatusUp {
				return fmt.Errorf("%s is still %v", name, health), nil
			}
		}

		return nil, nil
	})
}

// Cleanup removes all cluster-wide resources created by Deploy(). It notably does
// not remove resources inside the namespace given to Deploy(), as the namespace is
// owned by the calling code and we rely on its deletion (the namespace, not the code)
// for the cleanup.
func Cleanup(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	cfg *kubermaticv1.KubermaticConfiguration,
	timeout time.Duration,
) error {
	objects := []ctrlruntimeclient.Object{
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeportproxy.ClusterRoleBindingName(cfg),
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeportproxy.ClusterRoleName(cfg),
			},
		},
	}

	return wait.PollLog(ctx, log, 10*time.Second, timeout, func(ctx context.Context) (transient error, terminal error) {
		for _, obj := range objects {
			if err := client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete %s: %w", obj.GetName(), err), nil
			}
		}

		return nil, nil
	})
}
