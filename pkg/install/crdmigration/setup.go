/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package crdmigration

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"k8c.io/kubermatic/v2/pkg/install/util"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func InstallCRDs(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	// remove master cluster resources
	if err := installCRDsInCluster(ctx, logger.WithField("master", true), opt.MasterClient, opt.CRDDirectory); err != nil {
		return fmt.Errorf("processing the master cluster failed: %w", err)
	}

	// remove seed cluster resources
	for seedName, seedClient := range opt.SeedClients {
		if err := installCRDsInCluster(ctx, logger.WithField("seed", seedName), seedClient, opt.CRDDirectory); err != nil {
			return fmt.Errorf("processing the seed cluster failed: %w", err)
		}
	}

	return nil
}

func installCRDsInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client, src string) error {
	logger.Info("Installing new CRDs…")

	return util.DeployCRDs(ctx, client, logger, src)
}
