// +build ee

package provider

import (
	"context"

	eeprovider "github.com/kubermatic/kubermatic/api/pkg/ee/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, dcFile, namespace string, dynamicDatacenters bool) (SeedsGetter, error) {
	return eeprovider.SeedsGetterFactory(ctx, client, dcFile, namespace, dynamicDatacenters)
}

func seedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, seedName, dcFile, namespace string, dynamicDatacenters bool) (SeedGetter, error) {
	return eeprovider.SeedGetterFactory(ctx, client, seedName, dcFile, namespace, dynamicDatacenters)
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, kubeconfigFilePath string, dynamicDatacenters bool) (SeedKubeconfigGetter, error) {
	if dynamicDatacenters {
		return secretBasedSeedKubeconfigGetterFactory(ctx, client)
	}

	return eeprovider.KubeconfigBasedSeedKubeconfigGetterFactory(ctx, client, kubeconfigFilePath, dynamicDatacenters)
}
