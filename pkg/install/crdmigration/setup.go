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
	"path/filepath"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/util"
)

func InstallCRDs(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	logger.Info("Installing new CRDs…")

	hasOSM := opt.KubermaticConfiguration.Spec.FeatureGates.Has(features.OperatingSystemManager)
	osmCRDDirectory := filepath.Join(opt.CRDDirectory, "k8c.io", "operatingsystemmanager")
	masterLogger := logger.WithField("master", true)

	// process master cluster
	if err := util.DeployCRDs(ctx, opt.MasterClient, masterLogger, opt.CRDDirectory); err != nil {
		return fmt.Errorf("processing the master cluster failed: %w", err)
	}

	if hasOSM {
		if err := util.DeployCRDs(ctx, opt.MasterClient, masterLogger, osmCRDDirectory); err != nil {
			return fmt.Errorf("processing the master cluster failed: %w", err)
		}
	}

	// process seed clusters
	for seedName, seedClient := range opt.SeedClients {
		if err := util.DeployCRDs(ctx, seedClient, logger.WithField("seed", seedName), opt.CRDDirectory); err != nil {
			return fmt.Errorf("processing the seed cluster failed: %w", err)
		}

		if hasOSM {
			if err := util.DeployCRDs(ctx, seedClient, logger.WithField("seed", seedName), osmCRDDirectory); err != nil {
				return fmt.Errorf("processing the seed cluster failed: %w", err)
			}
		}
	}

	return nil
}
