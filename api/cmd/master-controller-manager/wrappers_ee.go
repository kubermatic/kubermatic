// +build ee

package main

import (
	"context"
	"flag"

	"go.uber.org/zap"

	eemasterctrlmgr "github.com/kubermatic/kubermatic/api/pkg/ee/cmd/master-controller-manager"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func addFlags(fs *flag.FlagSet) {
	eemasterctrlmgr.AddFlags(fs)
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (provider.SeedsGetter, error) {
	return eemasterctrlmgr.SeedsGetterFactory(ctx, client, namespace)
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt controllerRunOptions) (provider.SeedKubeconfigGetter, error) {
	return eemasterctrlmgr.SeedKubeconfigGetterFactory(ctx, client, opt.kubeconfig)
}

func setupSeedValidationWebhook(ctx context.Context, mgr manager.Manager, log *zap.SugaredLogger, opt controllerRunOptions, ctrlCtx *controllerContext) error {
	return eemasterctrlmgr.SetupSeedValidationWebhook(ctx, mgr, log, opt.seedvalidationHook, ctrlCtx.namespace, ctrlCtx.seedsGetter, ctrlCtx.seedKubeconfigGetter, opt.kubeconfig, opt.workerName)
}

func runMigrations(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, opt controllerRunOptions, ctrlCtx *controllerContext) error {
	return eemasterctrlmgr.RunMigrations(ctx, client, log, ctrlCtx.namespace, opt.kubeconfig)
}
