package provider

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/pkg/log"
	"github.com/kubermatic/kubermatic/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
func SeedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName string, namespace string) (SeedGetter, error) {
	return func() (*kubermaticv1.Seed, error) {
		seed := &kubermaticv1.Seed{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: seedName}, seed); err != nil {
			// allow callers to handle this gracefully
			if kerrors.IsNotFound(err) {
				return nil, err
			}

			return nil, fmt.Errorf("failed to get seed %q: %v", seedName, err)
		}

		seed.SetDefaults()

		return seed, nil
	}, nil
}

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (SeedsGetter, error) {
	return func() (map[string]*kubermaticv1.Seed, error) {
		seed := &kubermaticv1.Seed{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: defaultSeedName}, seed); err != nil {
			if kerrors.IsNotFound(err) {
				return nil, err
			}

			return nil, fmt.Errorf("failed to get seed %q: %v", defaultSeedName, err)
		}

		seed.SetDefaults()

		return map[string]*kubermaticv1.Seed{
			defaultSeedName: seed,
		}, nil
	}, nil
}

func SeedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client) (SeedKubeconfigGetter, error) {
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
