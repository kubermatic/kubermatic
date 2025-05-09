/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package kubernetesdashboard

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Migrate will remove resources no longer needed after the dashboard upgrade in KKP 2.27.
// This function can be removed in KKP 2.28.
func Migrate(ctx context.Context, client ctrlruntimeclient.Client, namespace string) error {
	if err := removeResource(ctx, client, &appsv1.Deployment{}, types.NamespacedName{Name: "dashboard-metrics-scraper", Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to delete old Deployment: %w", err)
	}

	if err := removeResource(ctx, client, &corev1.Service{}, types.NamespacedName{Name: "dashboard-metrics-scraper", Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to delete old Service: %w", err)
	}

	if err := removeResource(ctx, client, &rbacv1.Role{}, types.NamespacedName{Name: "system:kubernetes-dashboard", Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to delete old Role: %w", err)
	}

	if err := removeResource(ctx, client, &rbacv1.RoleBinding{}, types.NamespacedName{Name: "system:kubernetes-dashboard", Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to delete old RoleBinding: %w", err)
	}

	if err := removeResource(ctx, client, &corev1.ServiceAccount{}, types.NamespacedName{Name: "dashboard-metrics-scraper", Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to delete old ServiceAccount: %w", err)
	}

	if err := removeResource(ctx, client, &corev1.Secret{}, types.NamespacedName{Name: "kubernetes-dashboard-csrf", Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to delete old CSRF Secret: %w", err)
	}

	if err := removeResource(ctx, client, &corev1.Secret{}, types.NamespacedName{Name: "kubernetes-dashboard-key-holder", Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to delete old keyholder Secret: %w", err)
	}

	return nil
}

func removeResource(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, key types.NamespacedName) error {
	// Get the resource first to make use of the cache and skip unnecessary delete calls.
	if err := client.Get(ctx, key, obj); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return err
	} else if err != nil {
		return nil
	}

	return client.Delete(ctx, obj)
}
