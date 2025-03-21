//go:build !ee

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

	seedcontrollerlifecycle "k8c.io/kubermatic/v2/pkg/controller/shared/seed-controller-lifecycle"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func addFlags(fs *flag.FlagSet) {
	// NOP
}

func setupControllers(ctrlCtx *controllerContext) error {
	// NOP, no CE-only controllers exist
	return nil
}

func setupLifecycleControllerCreators(ctrlCtx *controllerContext) []seedcontrollerlifecycle.ControllerFactory {
	// NOP, no CE-only controllers exist
	return []seedcontrollerlifecycle.ControllerFactory{}
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (provider.SeedsGetter, error) {
	return kubernetes.SeedsGetterFactory(ctx, client, namespace)
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt controllerRunOptions) (provider.SeedKubeconfigGetter, error) {
	return kubernetes.SeedKubeconfigGetterFactory(ctx, client)
}

func resourceQuotaSynchronizerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(_ context.Context, _ manager.Manager, _ map[string]manager.Manager) (string, error) {
		return "", nil
	}
}

func resourceQuotaControllerFactoryCreator(ctrlCtx *controllerContext) seedcontrollerlifecycle.ControllerFactory {
	return func(_ context.Context, _ manager.Manager, _ map[string]manager.Manager) (string, error) {
		return "", nil
	}
}
