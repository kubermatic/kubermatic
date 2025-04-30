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

// In 2.26, the azure-file and azure-disk CSI implementations have been replaced
// with the upstream Helm charts, which has different labels and so some components
// need to be deleted prior to upgrades.
type csiAzureHelmMigration struct {
	nopMigration
}

func (m *csiAzureHelmMigration) Targets(cluster *kubermaticv1.Cluster, addonName string) bool {
	return addonName == "csi" && cluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] && cluster.Spec.Cloud.Azure != nil
}

func (m *csiAzureHelmMigration) PreApply(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client, userclusterClient ctrlruntimeclient.Client) error {
	ns := metav1.NamespaceSystem

	dep := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "csi-azurefile-controller", Namespace: ns}
	if err := deleteObjectIfLabelMissing(ctx, log, userclusterClient, dep, key, func(o *appsv1.Deployment) *metav1.LabelSelector {
		return o.Spec.Selector
	}); err != nil {
		return err
	}

	ds := &appsv1.DaemonSet{}
	key = types.NamespacedName{Name: "csi-azurefile-node", Namespace: ns}
	if err := deleteObjectIfLabelMissing(ctx, log, userclusterClient, ds, key, func(o *appsv1.DaemonSet) *metav1.LabelSelector {
		return o.Spec.Selector
	}); err != nil {
		return err
	}

	return nil
}

func deleteObjectIfLabelMissing[T ctrlruntimeclient.Object](
	ctx context.Context,
	log *zap.SugaredLogger,
	userclusterClient ctrlruntimeclient.Client,
	obj T,
	key types.NamespacedName,
	selectorGetter func(obj T) *metav1.LabelSelector,
) error {
	if err := userclusterClient.Get(ctx, key, obj); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}

	selector := selectorGetter(obj)
	if _, exists := selector.MatchLabels["app.kubernetes.io/name"]; !exists {
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		log.Infof("Deleting Azure %s %s to allow upgrade", kind, obj.GetName())

		if err := userclusterClient.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete %s: %w", kind, err)
		}
	}

	return nil
}
