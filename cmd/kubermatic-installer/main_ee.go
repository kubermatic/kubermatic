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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	eeinstaller "k8c.io/kubermatic/v2/pkg/ee/cmd/kubermatic-installer"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addCommands(cmd *cobra.Command, logger *logrus.Logger, versions kubermaticversion.Versions) {
	cmd.AddCommand(
		ConvertKubeconfigCommand(logger),
		DeployCommand(logger, versions),
		PrintCommand(),
		VersionCommand(logger, versions),
		MirrorImagesCommand(logger, versions),
		LocalCommand(logger),
	)
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client) (provider.SeedsGetter, error) {
	return eeinstaller.SeedsGetterFactory(ctx, client, kubermaticmaster.KubermaticOperatorNamespace)
}

func seedKubeconfigGetterFactory(ctx context.Context, client ctrlruntimeclient.Client) (provider.SeedKubeconfigGetter, error) {
	return eeinstaller.SeedKubeconfigGetterFactory(ctx, client)
}

// flags to be only used in EE edition.
func wrapDeployFlags(flagset *pflag.FlagSet, opt *DeployOptions) {
	flagset.BoolVar(&opt.DeployDefaultAppCatalog, "deploy-default-app-catalog", false, "Reconcile the default Application Catalog (EE only)")
}
