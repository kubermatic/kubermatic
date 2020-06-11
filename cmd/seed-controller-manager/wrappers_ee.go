// +build ee

package main

import (
	"context"
	"flag"

	eeseedctrlmgr "github.com/kubermatic/kubermatic/api/pkg/ee/cmd/seed-controller-manager"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addFlags(fs *flag.FlagSet) {
	eeseedctrlmgr.AddFlags(fs)
}

func seedGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, options controllerRunOptions) (provider.SeedGetter, error) {
	return eeseedctrlmgr.SeedGetterFactory(ctx, client, options.dc, options.namespace)
}
