/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package kubernetes

import (
	"bytes"
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SeedProvider struct that holds required components in order seeds.
type SeedProvider struct {
	clientPrivileged ctrlruntimeclient.Client
}

var _ provider.SeedProvider = &SeedProvider{}

func NewSeedProvider(client ctrlruntimeclient.Client) *SeedProvider {
	return &SeedProvider{
		clientPrivileged: client,
	}
}

func (p *SeedProvider) UpdateUnsecured(ctx context.Context, seed *kubermaticv1.Seed) (*kubermaticv1.Seed, error) {
	if err := p.clientPrivileged.Update(ctx, seed); err != nil {
		return nil, err
	}
	return seed, nil
}

func (p *SeedProvider) CreateUnsecured(ctx context.Context, seed *kubermaticv1.Seed) (*kubermaticv1.Seed, error) {
	if err := p.clientPrivileged.Create(ctx, seed); err != nil {
		return nil, err
	}
	return seed, nil
}

func (p *SeedProvider) CreateOrUpdateKubeconfigSecretForSeed(ctx context.Context, seed *kubermaticv1.Seed, kubeconfig []byte) error {
	kubeconfigRef, err := p.ensureKubeconfigSecret(ctx, seed, map[string][]byte{
		resources.KubeconfigSecretKey: kubeconfig,
	})
	if err != nil {
		return err
	}
	seed.Spec.Kubeconfig = *kubeconfigRef
	return nil
}

func (p *SeedProvider) ensureKubeconfigSecret(ctx context.Context, seed *kubermaticv1.Seed, secretData map[string][]byte) (*corev1.ObjectReference, error) {
	name := fmt.Sprintf("kubeconfig-%s", seed.Name)

	namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: name}
	existingSecret := &corev1.Secret{}

	if err := p.clientPrivileged.Get(ctx, namespacedName, existingSecret); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to probe for secret %q: %w", name, err)
		}
		return createSeedKubeconfigSecret(ctx, p.clientPrivileged, name, secretData)
	}

	return updateSeedKubeconfigSecret(ctx, p.clientPrivileged, existingSecret, secretData)
}

func createSeedKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, name string, secretData map[string][]byte) (*corev1.ObjectReference, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretData,
	}
	if err := client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig secret: %w", err)
	}

	return &corev1.ObjectReference{
		Kind:            secret.Kind,
		Namespace:       resources.KubermaticNamespace,
		Name:            secret.Name,
		UID:             secret.UID,
		APIVersion:      secret.APIVersion,
		ResourceVersion: secret.ResourceVersion,
	}, nil
}

func updateSeedKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, existingSecret *corev1.Secret, secretData map[string][]byte) (*corev1.ObjectReference, error) {
	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}

	requiresUpdate := false

	for k, v := range secretData {
		if !bytes.Equal(v, existingSecret.Data[k]) {
			requiresUpdate = true
			break
		}
	}

	if requiresUpdate {
		existingSecret.Data = secretData
		if err := client.Update(ctx, existingSecret); err != nil {
			return nil, fmt.Errorf("failed to update kubeconfig secret: %w", err)
		}
	}

	return &corev1.ObjectReference{
		Kind:            existingSecret.Kind,
		Namespace:       resources.KubermaticNamespace,
		Name:            existingSecret.Name,
		UID:             existingSecret.UID,
		APIVersion:      existingSecret.APIVersion,
		ResourceVersion: existingSecret.ResourceVersion,
	}, nil
}
