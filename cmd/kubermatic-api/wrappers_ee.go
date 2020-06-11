// +build ee

package main

import (
	"context"
	"flag"

	eeapi "github.com/kubermatic/kubermatic/api/pkg/ee/cmd/kubermatic-api"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addFlags(fs *flag.FlagSet) {
	eeapi.AddFlags(fs)
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt serverRunOptions) (provider.SeedsGetter, error) {
	return eeapi.SeedsGetterFactory(ctx, client, opt.namespace)
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt serverRunOptions) (provider.SeedKubeconfigGetter, error) {
	return eeapi.SeedKubeconfigGetterFactory(ctx, client, opt.kubeconfig)
}
