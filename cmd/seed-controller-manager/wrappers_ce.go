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

func seedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, options controllerRunOptions) (provider.SeedGetter, error) {
	return provider.SeedGetterFactory(ctx, client, options.dc, options.namespace)
}
