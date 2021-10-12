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
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/install/crdmigration"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/provider"

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
)

func MigrateCRDsCommand(logger *logrus.Logger) cli.Command {
	return cli.Command{
		Name:   "migrate-crds",
		Usage:  "Migrates the KKP CRDs to their new API groups",
		Action: MigrateCRDsAction(logger),
		Flags: []cli.Flag{
			migrateCRDsKubeContextFlag,
		},
	}
}

func MigrateCRDsAction(logger *logrus.Logger) cli.ActionFunc {
	return handleErrors(logger, setupLogger(logger, func(ctx *cli.Context) error {
		appContext := context.Background()
		namespace := kubermaticmaster.KubermaticOperatorNamespace

		//////////////////////////////////////
		// phase 0: preparations

		// get kube client to master cluster
		kubeContext := ctx.String(deployKubeContextFlag.Name)

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
		}

		//////////////////////////////////////
		// phase 1: preflight checks

		if err := crdmigration.PerformPreflightChecks(appContext, logger.WithField("phase", "preflight"), &opt); err != nil {
			return fmt.Errorf("preflight checks failed: %w", err)
		}

		//////////////////////////////////////
		// phase 2: backups

		// task 2.1: create a backup of all *.k8s.io KKP resources, one archive per seed cluster

		//////////////////////////////////////
		// phase 3: magic!

		// task 3.1: create new KKP CRDs
		// task 3.2: create copies of all kkp resources, but using the new API groups
		// task 3.3: remove ownerReferences from all old KKP resources
		// task 3.4: remove finalizers from all old KKP resources
		// task 3.5: remove all old KKP resources

		logger.Info("All Done")

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
		return nil, fmt.Errorf("failed to add scheme: %v", err)
	}

	if err := operatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("failed to add scheme: %v", err)
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
