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

	eeseedctrlmgr "k8c.io/kubermatic/v2/pkg/ee/cmd/seed-controller-manager"
	groupprojectbindingcontroller "k8c.io/kubermatic/v2/pkg/ee/group-project-binding/controller"
	resourcequotaseedcontroller "k8c.io/kubermatic/v2/pkg/ee/resource-quota/seed-controller"
	"k8c.io/kubermatic/v2/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addFlags(fs *flag.FlagSet) {
	// NOP
}

func seedGetterFactory(ctx context.Context, client ctrlruntimeclient.Reader, options controllerRunOptions) (provider.SeedGetter, error) {
	return eeseedctrlmgr.SeedGetterFactory(ctx, client, options.seedName, options.namespace)
}

func setupControllers(ctrlCtx *controllerContext) error {
	if err := resourcequotaseedcontroller.Add(ctrlCtx.mgr, ctrlCtx.log, ctrlCtx.runOptions.workerName, ctrlCtx.runOptions.workerCount); err != nil {
		return fmt.Errorf("failed to create resource quota controller: %w", err)
	}

	if err := groupprojectbindingcontroller.Add(ctrlCtx.ctx, ctrlCtx.mgr, ctrlCtx.log, ctrlCtx.runOptions.workerCount, false); err != nil {
		return fmt.Errorf("failed to create GroupProjectBinding controller: %w", err)
	}

	return nil
}
