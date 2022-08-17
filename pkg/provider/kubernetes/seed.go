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
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
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

	creators := []reconciling.NamedSecretCreatorGetter{
		seedKubeconfigSecretCreatorGetter(name, secretData),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, seed.Namespace, p.clientPrivileged); err != nil {
		return nil, err
	}

	return &corev1.ObjectReference{
		Kind:      "Secret",
		Namespace: seed.Namespace,
		Name:      name,
	}, nil
}

func seedKubeconfigSecretCreatorGetter(name string, secretData map[string][]byte) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return name, func(existing *corev1.Secret) (*corev1.Secret, error) {
			existing.Data = secretData
			return existing, nil
		}
	}
}
