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

package migrations

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// In KKP 2.26, the kube-state-metrics addon was switched to using the upstream Helm chart and
// due to labelling changes, some objects need to be removed before an upgrade can happen.
// https://github.com/kubermatic/kubermatic/pull/13599
type kubeStateMetricsMigration struct {
	nopMigration
}

func (m *kubeStateMetricsMigration) Targets(cluster *kubermaticv1.Cluster, addonName string) bool {
	return addonName == "kube-state-metrics"
}

func (m *kubeStateMetricsMigration) PreApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	dep := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "kube-state-metrics", Namespace: metav1.NamespaceSystem}

	if err := userclusterClient.Get(ctx, key, dep); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}

	if _, exists := dep.Spec.Selector.MatchLabels["app.kubernetes.io/instance"]; !exists {
		log.Infof("Deleting kube-state-metrics Deployment to allow upgrade")
		if err := userclusterClient.Delete(ctx, dep); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete Deployment: %w", err)
		}
	}

	return nil
}
