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
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	legacykubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func commands(logger *logrus.Logger, versions kubermaticversion.Versions) []cli.Command {
	return []cli.Command{
		VersionCommand(logger, versions),
		DeployCommand(logger, versions),
		PrintCommand(),
		ConvertKubeconfigCommand(logger),
		PreflightChecksCommand(logger),
		ShutdownCommand(logger),
		MigrateCRDsCommand(logger),
	}
}

func flags() []cli.Flag {
	return []cli.Flag{
		verboseFlag,
		chartsDirectoryFlag,
	}
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client) (provider.SeedsGetter, error) {
	return provider.SeedsGetterFactory(ctx, client, kubermaticmaster.KubermaticOperatorNamespace)
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client) (provider.SeedKubeconfigGetter, error) {
	return provider.SeedKubeconfigGetterFactory(ctx, client)
}

func getLegacySeeds(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (map[string]*legacykubermaticv1.Seed, error) {
	seed := &legacykubermaticv1.Seed{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: provider.DefaultSeedName}, seed); err != nil {
		if kerrors.IsNotFound(err) {
			// We should not fail if no seed exists and just return an
			// empty map.
			return map[string]*legacykubermaticv1.Seed{}, nil
		}

		return nil, fmt.Errorf("failed to get seed %q: %w", provider.DefaultSeedName, err)
	}

	seed.SetDefaults()

	return map[string]*legacykubermaticv1.Seed{
		provider.DefaultSeedName: seed,
	}, nil
}
