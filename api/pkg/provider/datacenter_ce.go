// +build !ee

package provider

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, kubeconfigFilePath string, dynamicDatacenters bool) (SeedKubeconfigGetter, error) {
	return secretBasedSeedKubeconfigGetterFactory(ctx, client)
}
