//go:build ee

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

	allowedregistrycontroller "k8c.io/kubermatic/v2/pkg/ee/allowed-registry-controller"
	eemasterctrlmgr "k8c.io/kubermatic/v2/pkg/ee/cmd/master-controller-manager"
	"k8c.io/kubermatic/v2/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addFlags(fs *flag.FlagSet) {
	// NOP
}

func setupControllers(ctrlCtx *controllerContext) error {
	if err := allowedregistrycontroller.Add(ctrlCtx.mgr, ctrlCtx.log, 1, ctrlCtx.namespace); err != nil {
		return fmt.Errorf("failed to create allowedregistry controller: %w", err)
	}

	return nil
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (provider.SeedsGetter, error) {
	return eemasterctrlmgr.SeedsGetterFactory(ctx, client, namespace)
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, opt controllerRunOptions) (provider.SeedKubeconfigGetter, error) {
	return eemasterctrlmgr.SeedKubeconfigGetterFactory(ctx, client)
}
