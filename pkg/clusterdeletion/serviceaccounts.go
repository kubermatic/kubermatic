/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// cleanupServiceAccounts removes ServiceAccounts created outside the cluster namespace.
// Specifically, it removes the etcd-launcher ServiceAccount from the kube-system namespace.
// This cleanup is part of the namespace cleanup finalizer since these resources are cluster-wide
// and created during cluster setup.
func (d *Deletion) cleanupServiceAccounts(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Delete etcd-launcher ServiceAccount from kube-system namespace
	saName := fmt.Sprintf("%s-%s", rbac.EtcdLauncherServiceAccountName, cluster.Name)
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: metav1.NamespaceSystem,
		},
	}

	err := d.seedClient.Delete(ctx, sa)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete ServiceAccount %s/%s: %w", metav1.NamespaceSystem, saName, err)
	}

	if err == nil {
		d.recorder.Eventf(cluster, nil, corev1.EventTypeNormal, "ServiceAccountCleanup", "Reconciling", "Deleted ServiceAccount %s from namespace %s", saName, metav1.NamespaceSystem)
	}

	return nil
}
