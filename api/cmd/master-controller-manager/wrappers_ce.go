// +build !ee

/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	seedvalidation "github.com/kubermatic/kubermatic/api/pkg/validation/seed"

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
	// Creates a new default validator
	validator, err := seedvalidation.NewDefaultSeedValidator(
		opt.workerName,
		ctrlCtx.seedsGetter,
		provider.SeedClientGetterFactory(ctrlCtx.seedKubeconfigGetter),
	)
	if err != nil {
		return fmt.Errorf("failed to create seed validator webhook server: %v", err)
	}
	server, err := opt.seedvalidationHook.Server(
		ctx,
		log,
		ctrlCtx.namespace,
		seedvalidation.CombineSeedValidateFuncs(
			// Combine the default validator with the one ensuring a single
			// seed can be created.
			// TODO(irozzo) add the test in the controlle, otherwise the check
			// can be easily bypassed without recompiling the code.
			seedvalidation.SingleSeedValidateFunc(ctrlCtx.namespace),
			validator.Validate,
		),
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
