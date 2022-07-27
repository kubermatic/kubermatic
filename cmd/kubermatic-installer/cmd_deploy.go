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
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/stack/common"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	kubermaticseed "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-seed"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/edition"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	MinHelmVersion = semverlib.MustParse("v3.0.0")
)

type DeployOptions struct {
	Options

	Config string

	Kubeconfig  string
	KubeContext string

	HelmBinary       string
	HelmValues       string
	HelmTimeout      time.Duration
	SkipDependencies bool
	Force            bool

	StorageClass     string
	DisableTelemetry bool

	MigrateCertManager         bool
	MigrateUpstreamCertManager bool
	MigrateNginx               bool
	MigrateOpenstackCSI        bool
	MigrateLogrotate           bool
}

func DeployCommand(logger *logrus.Logger, versions kubermaticversion.Versions) *cobra.Command {
	opt := DeployOptions{
		HelmTimeout: 5 * time.Minute,
		HelmBinary:  "helm",
	}

	cmd := &cobra.Command{
		Use:          "deploy [kubermatic-master | kubermatic-seed]",
		Short:        "Install or upgrade the current installation to the installer's built-in version",
		Long:         "Installs or upgrades the current installation to the installer's built-in version",
		RunE:         DeployFunc(logger, versions, &opt),
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			options.CopyInto(&opt.Options)

			if opt.Config == "" {
				opt.Config = os.Getenv("CONFIG_YAML")
			}
			if opt.Kubeconfig == "" {
				opt.Kubeconfig = os.Getenv("KUBECONFIG")
			}
			if opt.KubeContext == "" {
				opt.KubeContext = os.Getenv("KUBE_CONTEXT")
			}
			if opt.HelmValues == "" {
				opt.HelmValues = os.Getenv("HELM_VALUES")
			}
			if opt.HelmBinary == "" {
				opt.HelmBinary = os.Getenv("HELM_BINARY")
			}
		},
	}

	cmd.PersistentFlags().StringVar(&opt.Config, "config", "", "full path to the KubermaticConfiguration YAML file (only required during first installation, on upgrades the configuration can automatically be read from the cluster instead)")
	cmd.PersistentFlags().StringVar(&opt.Kubeconfig, "kubeconfig", "", "full path to where a kubeconfig with cluster-admin permissions for the target cluster")
	cmd.PersistentFlags().StringVar(&opt.KubeContext, "kube-context", "", "context to use from the given kubeconfig")

	cmd.PersistentFlags().StringVar(&opt.HelmValues, "helm-values", "", "full path to the Helm values.yaml used for customizing all charts")
	cmd.PersistentFlags().DurationVar(&opt.HelmTimeout, "helm-timeout", opt.HelmTimeout, "time to wait for Helm operations to finish")
	cmd.PersistentFlags().StringVar(&opt.HelmBinary, "helm-binary", opt.HelmBinary, "full path to the Helm 3 binary to use")
	cmd.PersistentFlags().BoolVar(&opt.SkipDependencies, "skip-dependencies", false, "skip pulling Helm chart dependencies (requires chart dependencies to be already downloaded)")
	cmd.PersistentFlags().BoolVar(&opt.Force, "force", false, "perform Helm upgrades even when the release is up-to-date")

	cmd.PersistentFlags().StringVar(&opt.StorageClass, "storageclass", "", fmt.Sprintf("type of StorageClass to create (one of %v)", common.SupportedStorageClassProviders().List()))
	cmd.PersistentFlags().BoolVar(&opt.DisableTelemetry, "disable-telemetry", false, "disable telemetry agents")

	cmd.PersistentFlags().BoolVar(&opt.MigrateCertManager, "migrate-cert-manager", false, "enable the migration for cert-manager CRDs from v1alpha2 to v1")
	cmd.PersistentFlags().BoolVar(&opt.MigrateUpstreamCertManager, "migrate-upstream-cert-manager", false, "enable the migration for cert-manager to chart version 2.1.0+")
	cmd.PersistentFlags().BoolVar(&opt.MigrateNginx, "migrate-upstream-nginx-ingress", false, "enable the migration procedure for nginx-ingress-controller (upgrade from v1.3.0+)")
	cmd.PersistentFlags().BoolVar(&opt.MigrateOpenstackCSI, "migrate-openstack-csidrivers", false, "(kubermatic-seed only) enable the data migration of CSIDriver of openstack user-clusters")
	cmd.PersistentFlags().BoolVar(&opt.MigrateLogrotate, "migrate-logrotate", false, "enable the data migration to delete the logrotate addon")

	return cmd
}

func DeployFunc(logger *logrus.Logger, versions kubermaticversion.Versions, opt *DeployOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		fields := logrus.Fields{
			"version": versions.Kubermatic,
			"edition": edition.KubermaticEdition,
		}
		if opt.Verbose {
			fields["git"] = versions.KubermaticCommit
		}

		// error out early if there is no useful Helm binary
		helmClient, err := helm.NewCLI(opt.HelmBinary, opt.Kubeconfig, opt.KubeContext, opt.HelmTimeout, logger)
		if err != nil {
			return fmt.Errorf("failed to create Helm client: %w", err)
		}

		helmVersion, err := helmClient.Version()
		if err != nil {
			return fmt.Errorf("failed to check Helm version: %w", err)
		}

		if helmVersion.LessThan(MinHelmVersion) {
			return fmt.Errorf(
				"the installer requires Helm >= %s, but detected %q as %s (use --helm-binary or $HELM_BINARY to override)",
				MinHelmVersion,
				opt.HelmBinary,
				helmVersion,
			)
		}

		stackName := ""
		if len(args) > 0 {
			stackName = args[0]
		}

		var kubermaticStack stack.Stack
		switch stackName {
		case "kubermatic-seed":
			kubermaticStack = kubermaticseed.NewStack()
		case "kubermatic-master", "":
			kubermaticStack = kubermaticmaster.NewStack()
		default:
			return fmt.Errorf("unknown stack %q specified", stackName)
		}

		logger.WithFields(fields).Info("🚀 Initializing installer…")

		// load config files
		if len(opt.Kubeconfig) == 0 {
			return errors.New("no kubeconfig (--kubeconfig or $KUBECONFIG) given")
		}

		// this can result in both configs being nil, if no --config is given
		kubermaticConfig, rawKubermaticConfig, err := loadKubermaticConfiguration(opt.Config)
		if err != nil {
			return fmt.Errorf("failed to load KubermaticConfiguration: %w", err)
		}

		helmValues, err := loadHelmValues(opt.HelmValues)
		if err != nil {
			return fmt.Errorf("failed to load Helm values: %w", err)
		}

		deployOptions := stack.DeployOptions{
			HelmClient:                         helmClient,
			HelmValues:                         helmValues,
			KubermaticConfiguration:            kubermaticConfig,
			RawKubermaticConfiguration:         rawKubermaticConfig,
			StorageClassProvider:               opt.StorageClass,
			ForceHelmReleaseUpgrade:            opt.Force,
			ChartsDirectory:                    opt.ChartsDirectory,
			EnableCertManagerV2Migration:       opt.MigrateCertManager,
			EnableCertManagerUpstreamMigration: opt.MigrateUpstreamCertManager,
			EnableNginxIngressMigration:        opt.MigrateNginx,
			EnableOpenstackCSIDriverMigration:  opt.MigrateOpenstackCSI,
			EnableLogrotateMigration:           opt.MigrateLogrotate,
			DisableTelemetry:                   opt.DisableTelemetry,
			DisableDependencyUpdate:            opt.SkipDependencies,
			Versions:                           versions,
		}

		// prepapre Kubernetes and Helm clients
		ctrlConfig, err := ctrlruntimeconfig.GetConfigWithContext(opt.KubeContext)
		if err != nil {
			return fmt.Errorf("failed to get config: %w", err)
		}

		mgr, err := manager.New(ctrlConfig, manager.Options{
			MetricsBindAddress:     "0",
			HealthProbeBindAddress: "0",
		})
		if err != nil {
			return fmt.Errorf("failed to construct mgr: %w", err)
		}

		// start the manager in its own goroutine
		appContext := context.Background()

		go func() {
			if err := mgr.Start(appContext); err != nil {
				logger.Fatalf("Failed to start Kubernetes client manager: %v", err)
			}
		}()

		// wait for caches to be synced
		mgrSyncCtx, cancel := context.WithTimeout(appContext, 30*time.Second)
		defer cancel()
		if synced := mgr.GetCache().WaitForCacheSync(mgrSyncCtx); !synced {
			logger.Fatal("Timed out while waiting for Kubernetes client caches to synchronize.")
		}

		kubeClient := mgr.GetClient()

		// try to auto-find the KubermaticConfiguration
		if kubermaticConfig == nil {
			kubermaticConfig, err = findKubermaticConfiguration(appContext, kubeClient, kubermaticmaster.KubermaticOperatorNamespace)
			if err != nil {
				return fmt.Errorf("failed to detect current KubermaticConfiguration: %w", err)
			}
		}

		// validate the configuration (in order to auto-fetch the config during upgrades,
		// this validation has to happen after we connected to the cluster)
		logger.Info("🚦 Validating the provided configuration…")

		subLogger := log.Prefix(logrus.NewEntry(logger), "   ")

		kubermaticConfig, helmValues, validationErrors := kubermaticStack.ValidateConfiguration(kubermaticConfig, helmValues, deployOptions, subLogger)
		if len(validationErrors) > 0 {
			logger.Error("⛔ The provided configuration files are invalid:")

			for _, e := range validationErrors {
				subLogger.Errorf("%v", e)
			}

			return errors.New("please review your configuration and try again")
		}

		logger.Info("✅ Provided configuration is valid.")

		if err := apiextensionsv1.AddToScheme(mgr.GetScheme()); err != nil {
			return fmt.Errorf("failed to add scheme: %w", err)
		}

		if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
			return fmt.Errorf("failed to add scheme: %w", err)
		}

		if err := certmanagerv1.AddToScheme(mgr.GetScheme()); err != nil {
			return fmt.Errorf("failed to add scheme: %w", err)
		}

		// prepare seed access components
		seedsGetter, err := seedsGetterFactory(appContext, kubeClient)
		if err != nil {
			return fmt.Errorf("failed to create Seeds getter: %w", err)
		}

		seedKubeconfigGetter, err := seedKubeconfigGetterFactory(appContext, kubeClient)
		if err != nil {
			return fmt.Errorf("failed to create Seed kubeconfig getter: %w", err)
		}

		deployOptions.KubermaticConfiguration = kubermaticConfig
		deployOptions.HelmValues = helmValues
		deployOptions.KubeClient = kubeClient
		deployOptions.Logger = subLogger
		deployOptions.SeedsGetter = seedsGetter
		deployOptions.SeedClientGetter = provider.SeedClientGetterFactory(seedKubeconfigGetter)

		logger.Info("🚦 Validating existing installation…")

		if errs := kubermaticStack.ValidateState(appContext, deployOptions); errs != nil {
			logger.Error("⛔ Cannot proceed with the installation:")

			for _, e := range errs {
				subLogger.Errorf("%v", e)
			}

			return errors.New("preflight checks have failed")
		}

		logger.Info("✅ Existing installation is valid.")

		logger.Infof("🛫 Deploying %s…", kubermaticStack.Name())

		if err := kubermaticStack.Deploy(appContext, deployOptions); err != nil {
			return err
		}

		logger.Infof("🛬 Installation completed successfully. %s", greeting())

		return nil
	})
}

func greeting() string {
	greetings := []string{
		"Have a nice day!",
		"Time for a break, maybe? ☺",
		"✌",
		"Thank you for using Kubermatic ❤",
	}

	return greetings[rand.Intn(len(greetings))]
}
