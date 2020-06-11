// +build !ee

package main

import (
	"context"

	"github.com/kubermatic/kubermatic/api/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt *controllerRunOptions) (provider.SeedsGetter, error) {
	return provider.SeedsGetterFactory(ctx, client, opt.namespace)
}
