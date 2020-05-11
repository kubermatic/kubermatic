// +build !ee

package provider

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func seedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName, dcFile, namespace string, dynamicDatacenters bool) (SeedGetter, error) {
	return func() (*kubermaticv1.Seed, error) {
		seed := &kubermaticv1.Seed{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: seedName}, seed); err != nil {
			// allow callers to handle this gracefully
			if kerrors.IsNotFound(err) {
				return nil, err
			}

			return nil, fmt.Errorf("failed to get seed %q: %v", seedName, err)
		}

		return seed, nil
	}, nil
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, dcFile, namespace string, dynamicDatacenters bool) (SeedsGetter, error) {
	return func() (map[string]*kubermaticv1.Seed, error) {
		seed := &kubermaticv1.Seed{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaultSeedName}, seed); err != nil {
			if kerrors.IsNotFound(err) {
				return nil, err
			}

			return nil, fmt.Errorf("failed to get seed %q: %v", defaultSeedName, err)
		}

		return map[string]*kubermaticv1.Seed{
			defaultSeedName: seed,
		}, nil
	}, nil
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, kubeconfigFilePath, namespace string, dynamicDatacenters bool) (SeedKubeconfigGetter, error) {
	return func(seed *kubermaticv1.Seed) (*rest.Config, error) {
		secret := &corev1.Secret{}
		name := types.NamespacedName{
			Namespace: seed.Spec.Kubeconfig.Namespace,
			Name:      seed.Spec.Kubeconfig.Name,
		}
		if err := client.Get(ctx, name, secret); err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig secret %q: %v", name.String(), err)
		}

		fieldPath := seed.Spec.Kubeconfig.FieldPath
		if len(fieldPath) == 0 {
			fieldPath = DefaultKubeconfigFieldPath
		}
		if _, exists := secret.Data[fieldPath]; !exists {
			return nil, fmt.Errorf("secret %q has no key %q", name.String(), fieldPath)
		}

		cfg, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[fieldPath])
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %v", err)
		}
		kubermaticlog.Logger.With("seed", seed.Name).Debug("Successfully got kubeconfig")
		return cfg, nil
	}, nil
}
