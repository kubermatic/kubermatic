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
	"k8c.io/kubermatic/v2/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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

var (
	// emptySeedMap is returned when the default seed is not present.
	emptySeedMap = map[string]*kubermaticv1.Seed{}
)

// SeedGetterFactory returns a SeedGetter. It has validation of all its arguments.
func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Reader, seedName string, namespace string) (provider.SeedGetter, error) {
	return func() (*kubermaticv1.Seed, error) {
		seed := &kubermaticv1.Seed{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: seedName}, seed); err != nil {
			// allow callers to handle this gracefully
			if apierrors.IsNotFound(err) {
				return nil, err
			}

			return nil, fmt.Errorf("failed to get seed %q: %w", seedName, err)
		}

		seed.SetDefaults()

		return seed, nil
	}, nil
}

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (provider.SeedsGetter, error) {
	return func() (map[string]*kubermaticv1.Seed, error) {
		seed := &kubermaticv1.Seed{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: provider.DefaultSeedName}, seed); err != nil {
			if apierrors.IsNotFound(err) {
				// We should not fail if no seed exists and just return an
				// empty map.
				return emptySeedMap, nil
			}

			return nil, fmt.Errorf("failed to get seed %q: %w", provider.DefaultSeedName, err)
		}

		seed.SetDefaults()

		return map[string]*kubermaticv1.Seed{
			provider.DefaultSeedName: seed,
		}, nil
	}, nil
}

func SeedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Reader) (provider.SeedKubeconfigGetter, error) {
	return func(seed *kubermaticv1.Seed) (*rest.Config, error) {
		secret := &corev1.Secret{}
		name := types.NamespacedName{
			Namespace: seed.Spec.Kubeconfig.Namespace,
			Name:      seed.Spec.Kubeconfig.Name,
		}
		if name.Namespace == "" {
			name.Namespace = seed.Namespace
		}
		if err := client.Get(ctx, name, secret); err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig secret %q: %w", name.String(), err)
		}

		fieldPath := seed.Spec.Kubeconfig.FieldPath
		if len(fieldPath) == 0 {
			fieldPath = provider.DefaultKubeconfigFieldPath
		}
		if _, exists := secret.Data[fieldPath]; !exists {
			return nil, fmt.Errorf("secret %q has no key %q", name.String(), fieldPath)
		}

		cfg, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[fieldPath])
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
		return cfg, nil
	}, nil
}

// SeedClientGetterFactory returns a SeedClientGetter. It uses a RestMapperCache to cache
// the discovery data, which considerably speeds up client creation.
func SeedClientGetterFactory(kubeconfigGetter provider.SeedKubeconfigGetter) provider.SeedClientGetter {
	cache := restmapper.New()
	return func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
		cfg, err := kubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		return cache.Client(cfg)
	}
}
