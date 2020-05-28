// +build ee

package main

import (
	"context"

	eeprovider "github.com/kubermatic/kubermatic/api/pkg/ee/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt *controllerRunOptions) (provider.SeedsGetter, error) {
	return eeprovider.SeedsGetterFactory(ctx, client, "", opt.namespace, true)
}
