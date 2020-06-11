// +build !ee

package main

import (
	"context"
	"flag"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func addFlags(fs *flag.FlagSet) {
	// NOP
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (provider.SeedsGetter, error) {
	return provider.SeedsGetterFactory(ctx, client, namespace)
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt controllerRunOptions) (provider.SeedKubeconfigGetter, error) {
	return provider.SeedKubeconfigGetterFactory(ctx, client)
}

func setupSeedValidationWebhook(ctx context.Context, mgr manager.Manager, log *zap.SugaredLogger, opt controllerRunOptions, ctrlCtx *controllerContext) error {
	server, err := opt.seedvalidationHook.Server(
		ctx,
		log,
		ctrlCtx.namespace,
		opt.workerName,
		ctrlCtx.seedsGetter,
		provider.SeedClientGetterFactory(ctrlCtx.seedKubeconfigGetter),
		false)
	if err != nil {
		return fmt.Errorf("failed to create seed validation webhook server: %v", err)
	}

	if err := mgr.Add(server); err != nil {
		return fmt.Errorf("failed to add the seed validation webhook to the mgr: %v", err)
	}

	return nil
}

func runMigrations(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, opt controllerRunOptions, ctrlCtx *controllerContext) error {
	// NOP
	return nil
}
