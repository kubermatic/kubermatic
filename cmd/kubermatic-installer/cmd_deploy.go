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
	userclustermla "k8c.io/kubermatic/v2/pkg/install/stack/usercluster-mla"
	"k8c.io/kubermatic/v2/pkg/log"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/edition"
	"k8c.io/kubermatic/v2/pkg/util/flagopts"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	MinHelmVersion            = semverlib.MustParse("v3.0.0")
	UserClusterMinHelmTimeout = 15 * time.Minute
)

type DeployOptions struct {
	Options

	Config string

	Kubeconfig  string
	KubeContext string

	HelmBinary         string
	HelmValues         string
	HelmTimeout        time.Duration
	SkipDependencies   bool
	SkipSeedValidation sets.Set[string]
	Force              bool

	StorageClass       string
	DisableTelemetry   bool
	AllowEditionChange bool

	MigrateCertManager         bool
	MigrateUpstreamCertManager bool
	MigrateNginx               bool

	MLASkipMinio             bool
	MLASkipMinioLifecycleMgr bool
	MLAForceMLASecrets       bool
	MLAIncludeIap            bool

	DeployDefaultAppCatalog bool

	SkipCharts []string
}

func DeployCommand(logger *logrus.Logger, versions kubermaticversion.Versions) *cobra.Command {
	opt := DeployOptions{
		HelmTimeout:        5 * time.Minute,
		HelmBinary:         "helm",
		SkipSeedValidation: sets.New[string](),
	}

	cmd := &cobra.Command{
		Use:          "deploy [kubermatic-master | kubermatic-seed | usercluster-mla]",
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
	cmd.PersistentFlags().Var(flagopts.SetFlag(opt.SkipSeedValidation), "skip-seed-validation", "comma-separated list of seed clusters to skip running the preflight checks on (use with caution, as this can lead to defunct KKP setups)")
	cmd.PersistentFlags().BoolVar(&opt.Force, "force", false, "perform Helm upgrades even when the release is up-to-date")

	cmd.PersistentFlags().StringVar(&opt.StorageClass, "storageclass", "", fmt.Sprintf("type of StorageClass to create (one of %v)", sets.List(common.SupportedStorageClassProviders())))
	cmd.PersistentFlags().BoolVar(&opt.DisableTelemetry, "disable-telemetry", false, "disable telemetry agents")
	cmd.PersistentFlags().BoolVar(&opt.AllowEditionChange, "allow-edition-change", false, "allow up- or downgrading between Community and Enterprise editions")

	cmd.PersistentFlags().BoolVar(&opt.MigrateCertManager, "migrate-cert-manager", false, "enable the migration for cert-manager CRDs from v1alpha2 to v1")
	cmd.PersistentFlags().BoolVar(&opt.MigrateUpstreamCertManager, "migrate-upstream-cert-manager", false, "enable the migration for cert-manager to chart version 2.1.0+")
	cmd.PersistentFlags().BoolVar(&opt.MigrateNginx, "migrate-upstream-nginx-ingress", false, "enable the migration procedure for nginx-ingress-controller (upgrade from v1.3.0+)")

	cmd.PersistentFlags().BoolVar(&opt.MLASkipMinio, "mla-skip-minio", false, "(UserCluster MLA) skip installation of UserCluster MLA Minio")
	cmd.PersistentFlags().BoolVar(&opt.MLASkipMinioLifecycleMgr, "mla-skip-minio-lifecycle-mgr", false, "(UserCluster MLA) skip installation of userCluster MLA Minio Bucket Lifecycle Manager")
	cmd.PersistentFlags().BoolVar(&opt.MLAForceMLASecrets, "mla-force-secrets", false, "(UserCluster MLA) force reinstallation of mla-secrets Helm chart")
	cmd.PersistentFlags().BoolVar(&opt.MLAIncludeIap, "mla-include-iap", false, "(UserCluster MLA) Include Identity-Aware Proxy installation")

	wrapDeployFlags(cmd.PersistentFlags(), &opt)

	cmd.PersistentFlags().StringSliceVar(&opt.SkipCharts, "skip-charts", nil, "skip helm chart deployment (some of cert-manager, nginx-ingress-controller, dex)")

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

		stackName := ""
		if len(args) > 0 {
			stackName = args[0]
		}
		if stackName == "usercluster-mla" && opt.HelmTimeout <= UserClusterMinHelmTimeout {
			logger.Infof("🚦️ For usercluster-mla deployment, it is recommended to use Helm timeout value of at least %v. Overriding the current value of %s.", UserClusterMinHelmTimeout, opt.HelmTimeout)
			opt.HelmTimeout = UserClusterMinHelmTimeout
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

		var kubermaticStack stack.Stack
		switch stackName {
		case "usercluster-mla":
			kubermaticStack = userclustermla.NewStack()
		case "kubermatic-seed":
			kubermaticStack = kubermaticseed.NewStack()
		case "kubermatic-master", "":
			kubermaticStack = kubermaticmaster.NewStack(true)
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
			DisableTelemetry:                   opt.DisableTelemetry,
			DisableDependencyUpdate:            opt.SkipDependencies,
			AllowEditionChange:                 opt.AllowEditionChange,
			MLASkipMinio:                       opt.MLASkipMinio,
			MLASkipMinioLifecycleMgr:           opt.MLASkipMinioLifecycleMgr,
			MLAForceSecrets:                    opt.MLAForceMLASecrets,
			MLAIncludeIap:                      opt.MLAIncludeIap,
			Versions:                           versions,
			SkipCharts:                         opt.SkipCharts,
			DeployDefaultAppCatalog:            opt.DeployDefaultAppCatalog,
			SkipSeedValidation:                 opt.SkipSeedValidation,
		}

		// prepapre Kubernetes and Helm clients
		ctrlConfig, err := ctrlruntimeconfig.GetConfigWithContext(opt.KubeContext)
		if err != nil {
			return fmt.Errorf("failed to get config: %w", err)
		}

		mgr, err := manager.New(ctrlConfig, manager.Options{
			Metrics:                metricsserver.Options{BindAddress: "0"},
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
		deployOptions.RestConfig = ctrlConfig
		deployOptions.Logger = subLogger
		deployOptions.SeedsGetter = seedsGetter
		deployOptions.SeedClientGetter = kubernetesprovider.SeedClientGetterFactory(seedKubeconfigGetter)

		logger.Info("🚦 Validating existing installation…")

		if errs := kubermaticStack.ValidateState(appContext, deployOptions); len(errs) > 0 {
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
