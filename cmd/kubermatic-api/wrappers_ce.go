// +build !ee

package main

import (
	"context"
	"flag"

	"github.com/kubermatic/kubermatic/api/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addFlags(fs *flag.FlagSet) {
	// NOP
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt serverRunOptions) (provider.SeedsGetter, error) {
	return provider.SeedsGetterFactory(ctx, client, opt.namespace)
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt serverRunOptions) (provider.SeedKubeconfigGetter, error) {
	return provider.SeedKubeconfigGetterFactory(ctx, client)
}
