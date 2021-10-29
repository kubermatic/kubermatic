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

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	newv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/install/crdmigration"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/provider"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	migrateCRDsKubeContextFlag = cli.StringFlag{
		Name:   "kube-context",
		Usage:  "Context to use from the given kubeconfig",
		EnvVar: "KUBE_CONTEXT",
	}
	keepOldResourcesFlag = cli.BoolFlag{
		Name:  "keep-resources",
		Usage: "Do not delete resources in the old API group when the migration is completed",
	}
)

func MigrateCRDsCommand(logger *logrus.Logger) cli.Command {
	return cli.Command{
		Name:   "migrate-crds",
		Usage:  "(development only) Migrates the KKP CRDs to their new API groups",
		Action: MigrateCRDsAction(logger),
		Hidden: true, // users must not run this before it's released
		Flags: []cli.Flag{
			migrateCRDsKubeContextFlag,
			keepOldResourcesFlag,
		},
	}
}

func MigrateCRDsAction(logger *logrus.Logger) cli.ActionFunc {
	return handleErrors(logger, setupLogger(logger, func(ctx *cli.Context) error {
		appContext := context.Background()
		namespace := kubermaticmaster.KubermaticOperatorNamespace

		// ////////////////////////////////////
		// phase 0: preparations

		// get kube client to master cluster
		kubeContext := ctx.String(migrateCRDsKubeContextFlag.Name)

		logger.Info("Creating Kubernetes client to the master cluster…")

		kubeClient, err := getKubeClient(appContext, logger, kubeContext)
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes client: %w", err)
		}

		// retrieve KubermaticConfiguration
		configGetter, err := provider.DynamicKubermaticConfigurationGetterFactory(kubeClient, namespace)
		if err != nil {
			return fmt.Errorf("failed to create KubermaticConfiguration client: %w", err)
		}

		config, err := configGetter(appContext)
		if err != nil {
			return fmt.Errorf("failed to retrieve KubermaticConfiguration: %w", err)
		}

		// find all seeds
		seedsGetter, err := seedsGetterFactory(appContext, kubeClient)
		if err != nil {
			return fmt.Errorf("failed to create Seeds getter: %w", err)
		}

		seedKubeconfigGetter, err := seedKubeconfigGetterFactory(appContext, kubeClient)
		if err != nil {
			return fmt.Errorf("failed to create Seed kubeconfig getter: %w", err)
		}

		logger.Info("Retrieving Seeds…")

		allSeeds, err := seedsGetter()
		if err != nil {
			return fmt.Errorf("failed to list Seeds: %w", err)
		}

		logger.Infof("Found %d Seeds.", len(allSeeds))

		// build kube client for each seed cluster
		seedClients := map[string]ctrlruntimeclient.Client{}
		seedClientGetter := provider.SeedClientGetterFactory(seedKubeconfigGetter)

		logger.Info("Creating Kubernetes client for each Seed…")

		for _, seed := range allSeeds {
			seedClient, err := seedClientGetter(seed)
			if err != nil {
				return fmt.Errorf("failed to create Kubernetes client for Seed %q: %w", seed.Name, err)
			}

			seedClients[seed.Name] = seedClient
		}

		// assemble migration options
		opt := crdmigration.Options{
			KubermaticNamespace:     namespace,
			KubermaticConfiguration: config,
			MasterClient:            kubeClient,
			Seeds:                   allSeeds,
			SeedClients:             seedClients,
			CRDDirectory:            filepath.Join(ctx.GlobalString(chartsDirectoryFlag.Name), "kubermatic-operator", "crd", "k8c.io"),
		}

		// ////////////////////////////////////
		// phase 1: preflight checks

		if err := crdmigration.PerformPreflightChecks(appContext, logger.WithField("phase", "preflight"), &opt); err != nil {
			return fmt.Errorf("preflight checks failed: %w", err)
		}

		// ////////////////////////////////////
		// phase 2: backups

		if err := crdmigration.CreateBackups(appContext, logger.WithField("phase", "backup"), &opt); err != nil {
			return fmt.Errorf("backups failed: %w", err)
		}

		// ////////////////////////////////////
		// phase 3: magic!

		if err := crdmigration.InstallCRDs(appContext, logger.WithField("phase", "setup"), &opt); err != nil {
			return fmt.Errorf("CRD setup failed: %w", err)
		}

		if err := crdmigration.DuplicateResources(appContext, logger.WithField("phase", "cloning"), &opt); err != nil {
			return fmt.Errorf("resource cloning failed: %w", err)
		}

		if !ctx.Bool(keepOldResourcesFlag.Name) {
			if err := crdmigration.RemoveOldResources(appContext, logger.WithField("phase", "cleanup"), &opt); err != nil {
				return fmt.Errorf("resource cleanup failed: %w", err)
			}
		}

		// ////////////////////////////////////
		// phase 4: time for cigars

		logger.Info("All Done :)")
		logger.Info("All KKP resources have been successfully migrated to the new API group.")

		if ctx.Bool(keepOldResourcesFlag.Name) {
			logger.Info("You can remove the resources from the old group, kubermatic.k8s.io, manually at a later time.")
		}

		logger.Info("Please run the `deploy` command now to update your KKP. The KKP Operator will reconcile and restart all controllers.")

		return nil
	}))
}

func getKubeClient(ctx context.Context, logger logrus.FieldLogger, kubeContext string) (ctrlruntimeclient.Client, error) {
	// prepapre Kubernetes and Helm clients
	ctrlConfig, err := ctrlruntimeconfig.GetConfigWithContext(kubeContext)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	mgr, err := manager.New(ctrlConfig, manager.Options{
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to construct mgr: %w", err)
	}

	if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("failed to add scheme: %w", err)
	}

	if err := newv1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("failed to add scheme: %w", err)
	}

	if err := operatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("failed to add scheme: %w", err)
	}

	if err := apiextensionsv1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("failed to add scheme: %w", err)
	}

	// start the manager in its own goroutine
	go func() {
		if err := mgr.Start(ctx); err != nil {
			logger.Fatalf("Failed to start Kubernetes client manager: %w", err)
		}
	}()

	// wait for caches to be synced
	mgrSyncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if synced := mgr.GetCache().WaitForCacheSync(mgrSyncCtx); !synced {
		logger.Fatal("Timed out while waiting for Kubernetes client caches to synchronize.")
	}

	return mgr.GetClient(), nil
}
