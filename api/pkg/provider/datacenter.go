package provider

import (
	"context"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/restmapper"

	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// defaultSeedName is the name of the Seed resource that is used
	// in the Community Edition, which is limited to a single seed.
	defaultSeedName = "kubermatic"
)

// SeedGetter is a function to retrieve a single seed
type SeedGetter = func() (*kubermaticv1.Seed, error)

// SeedsGetter is a function to retrieve a list of seeds
type SeedsGetter = func() (map[string]*kubermaticv1.Seed, error)

// SeedKubeconfigGetter is used to fetch the kubeconfig for a given seed
type SeedKubeconfigGetter = func(seed *kubermaticv1.Seed) (*rest.Config, error)

// SeedClientGetter is used to get a ctrlruntimeclient for a given seed
type SeedClientGetter = func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error)

// ClusterProviderGetter is used to get a clusterProvider
type ClusterProviderGetter = func(seed *kubermaticv1.Seed) (ClusterProvider, error)

// AddonProviderGetterr is used to get an AddonProvider
type AddonProviderGetter = func(seed *kubermaticv1.Seed) (AddonProvider, error)

// SeedGetterFactory returns a SeedGetter. It has validation of all its arguments
func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName, dcFile, namespace string, dynamicDatacenters bool) (SeedGetter, error) {
	seedGetter, err := seedGetterFactory(ctx, client, seedName, dcFile, namespace, dynamicDatacenters)
	if err != nil {
		return nil, err
	}
	return func() (*kubermaticv1.Seed, error) {
		seed, err := seedGetter()
		if err != nil {
			return nil, err
		}
		seed.SetDefaults()
		return seed, nil
	}, nil
}

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, dcFile, namespace string, dynamicDatacenters bool) (SeedsGetter, error) {
	seedsGetter, err := seedsGetterFactory(ctx, client, dcFile, namespace, dynamicDatacenters)
	if err != nil {
		return nil, err
	}
	return func() (map[string]*kubermaticv1.Seed, error) {
		seeds, err := seedsGetter()
		if err != nil {
			return nil, err
		}
		for idx := range seeds {
			seeds[idx].SetDefaults()
		}
		return seeds, nil
	}, nil
}

func SeedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, kubeconfigFilePath, namespace string, dynamicDatacenters bool) (SeedKubeconfigGetter, error) {
	return seedKubeconfigGetterFactory(ctx, client, kubeconfigFilePath, namespace, dynamicDatacenters)
}

// SeedClientGetterFactory returns a SeedClientGetter. It uses a RestMapperCache to cache
// the discovery data, which considerably speeds up client creation.
func SeedClientGetterFactory(kubeconfigGetter SeedKubeconfigGetter) SeedClientGetter {
	cache := restmapper.New()
	return func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
		cfg, err := kubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		return cache.Client(cfg)
	}
}
