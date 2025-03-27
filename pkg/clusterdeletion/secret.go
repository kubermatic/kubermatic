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

package clusterdeletion

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (d *Deletion) cleanupCredentialsSecrets(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if err := d.deleteSecret(ctx, cluster); err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, d.seedClient, cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer)
}

func (d *Deletion) deleteSecret(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	secretName := cluster.GetSecretName()
	if secretName == "" {
		return nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{Name: secretName, Namespace: resources.KubermaticNamespace}
	err := d.seedClient.Get(ctx, name, secret)
	// It's already gone
	if apierrors.IsNotFound(err) {
		return nil
	}

	// Something failed while loading the secret
	if err != nil {
		return fmt.Errorf("failed to get Secret %q: %w", name.String(), err)
	}

	if err := d.seedClient.Delete(ctx, secret); err != nil {
		return fmt.Errorf("failed to delete Secret %q: %w", name.String(), err)
	}

	// We successfully deleted the secret
	return nil
}
