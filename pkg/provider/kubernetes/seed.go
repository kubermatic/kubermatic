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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

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

func GetSeedKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed) (*corev1.Secret, error) {
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

	return secret, nil
}

func GetSeedKubeconfig(ctx context.Context, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed) ([]byte, error) {
	secret, err := GetSeedKubeconfigSecret(ctx, client, seed)
	if err != nil {
		return nil, err
	}

	fieldPath := seed.Spec.Kubeconfig.FieldPath
	if len(fieldPath) == 0 {
		fieldPath = provider.DefaultKubeconfigFieldPath
	}
	if _, exists := secret.Data[fieldPath]; !exists {
		return nil, fmt.Errorf("secret %q has no key %q", secret.Name, fieldPath)
	}

	return secret.Data[fieldPath], nil
}

func SeedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client) (provider.SeedKubeconfigGetter, error) {
	return func(seed *kubermaticv1.Seed) (*rest.Config, error) {
		kubeconfig, err := GetSeedKubeconfig(ctx, client, seed)
		if err != nil {
			return nil, err
		}

		cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
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
