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

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanUpCredentialsSecrets(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if err := d.deleteSecret(ctx, cluster); err != nil {
		return err
	}

	oldCluster := cluster.DeepCopy()
	kuberneteshelper.RemoveFinalizer(cluster, kubermaticapiv1.CredentialsSecretsCleanupFinalizer)
	return d.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func (d *Deletion) deleteSecret(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	secretName := cluster.GetSecretName()
	if secretName == "" {
		return nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{Name: secretName, Namespace: resources.KubermaticNamespace}
	err := d.seedClient.Get(ctx, name, secret)
	// Its already gone
	if kerrors.IsNotFound(err) {
		return nil
	}

	// Something failed while loading the secret
	if err != nil {
		return fmt.Errorf("failed to get Secret %q: %v", name.String(), err)
	}

	if err := d.seedClient.Delete(ctx, secret); err != nil {
		return fmt.Errorf("failed to delete Secret %q: %v", name.String(), err)
	}

	// We successfully deleted the secret
	return nil
}
