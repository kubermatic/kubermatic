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
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/stack/common"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	kubermaticseed "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-seed"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/edition"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	minHelmVersion = semver.MustParse("v3.0.0")

	deployForceFlag = cli.BoolFlag{
		Name:  "force",
		Usage: "Perform Helm upgrades even when the release is up-to-date",
	}
	deployConfigFlag = cli.StringFlag{
		Name:   "config",
		Usage:  "Full path to the KubermaticConfiguration YAML file",
		EnvVar: "CONFIG_YAML",
	}
	deployHelmValuesFlag = cli.StringFlag{
		Name:   "helm-values",
		Usage:  "Full path to the Helm values.yaml used for customizing all charts",
		EnvVar: "VALUES_YAML",
	}
	deployKubeconfigFlag = cli.StringFlag{
		Name:   "kubeconfig",
		Usage:  "Full path to where a kubeconfig with cluster-admin permissions for the target cluster",
		EnvVar: "KUBECONFIG",
	}
	deployKubeContextFlag = cli.StringFlag{
		Name:   "kube-context",
		Usage:  "Context to use from the given kubeconfig",
		EnvVar: "KUBE_CONTEXT",
	}
	deployHelmTimeoutFlag = cli.DurationFlag{
		Name:  "helm-timeout",
		Usage: "Time to wait for Helm operations to finish",
		Value: 5 * time.Minute,
	}
	deployHelmBinaryFlag = cli.StringFlag{
		Name:   "helm-binary",
		Usage:  "Full path to the Helm 3 binary to use",
		Value:  "helm",
		EnvVar: "HELM_BINARY",
	}
	deployStorageClassFlag = cli.StringFlag{
		Name:  "storageclass",
		Usage: fmt.Sprintf("Type of StorageClass to create (one of %v)", common.SupportedStorageClassProviders().List()),
	}
)

func DeployCommand(logger *logrus.Logger, versions kubermaticversion.Versions) cli.Command {
	return cli.Command{
		Name:      "deploy",
		Usage:     "Installs or upgrades the current installation to the installer's built-in version",
		Action:    DeployAction(logger, versions),
		ArgsUsage: "[STACK=kubermatic-master]",
		Flags: []cli.Flag{
			deployForceFlag,
			deployConfigFlag,
			deployHelmValuesFlag,
			deployKubeconfigFlag,
			deployKubeContextFlag,
			deployHelmTimeoutFlag,
			deployHelmBinaryFlag,
			deployStorageClassFlag,
		},
	}
}

type deployOptions struct {
	cli                 *cli.Context
	subLogger           *logrus.Entry
	helmClient          helm.Client
	kubeClient          ctrlruntimeclient.Client
	helmValues          *yamled.Document
	kubermaticConfig    *operatorv1alpha1.KubermaticConfiguration
	rawKubermaticConfig *unstructured.Unstructured
}

func DeployAction(logger *logrus.Logger, versions kubermaticversion.Versions) cli.ActionFunc {
	return handleErrors(logger, setupLogger(logger, func(ctx *cli.Context) error {
		fields := logrus.Fields{
			"version": versions.Kubermatic,
			"edition": edition.KubermaticEdition,
		}
		if ctx.GlobalBool("verbose") {
			fields["git"] = versions.KubermaticCommit
		}

		// error out early if there is no useful Helm binary
		kubeconfig := ctx.String(deployKubeconfigFlag.Name)
		kubeContext := ctx.String(deployKubeContextFlag.Name)
		helmTimeout := ctx.Duration(deployHelmTimeoutFlag.Name)
		helmBinary := ctx.String(deployHelmBinaryFlag.Name)

		helmClient, err := helm.NewCLI(helmBinary, kubeconfig, kubeContext, helmTimeout, logger)
		if err != nil {
			return fmt.Errorf("failed to create Helm client: %v", err)
		}

		helmVersion, err := helmClient.Version()
		if err != nil {
			return fmt.Errorf("failed to check Helm version: %v", err)
		}

		if helmVersion.LessThan(minHelmVersion) {
			return fmt.Errorf(
				"the installer requires Helm >= %s, but detected %q as %s (use --%s or $%s to override)",
				minHelmVersion,
				helmBinary,
				helmVersion,
				deployHelmBinaryFlag.Name,
				deployHelmBinaryFlag.EnvVar)
		}

		var kubermaticStack stack.Stack
		stackName := ctx.Args().First()
		switch stackName {
		case "kubermatic-seed":
			kubermaticStack = kubermaticseed.NewStack()

		case "kubermatic-master":
			fallthrough

		default:
			kubermaticStack = kubermaticmaster.NewStack()
		}

		if kubermaticStack == nil {
			return fmt.Errorf("unknown stack %q specified", stackName)
		}

		logger.WithFields(fields).Info("🛫 Initializing installer…")

		// load config files
		if len(kubeconfig) == 0 {
			return fmt.Errorf("no kubeconfig (--%s or $%s) given", deployKubeContextFlag.Name, deployKubeconfigFlag.EnvVar)
		}

		kubermaticConfig, rawKubermaticConfig, err := loadKubermaticConfiguration(ctx.String(deployConfigFlag.Name))
		if err != nil {
			return fmt.Errorf("failed to load KubermaticConfiguration: %v", err)
		}

		helmValues, err := loadHelmValues(ctx.String(deployHelmValuesFlag.Name))
		if err != nil {
			return fmt.Errorf("failed to load Helm values: %v", err)
		}

		// validate the configuration
		logger.Info("🚦 Validating the provided configuration…")

		subLogger := log.Prefix(logrus.NewEntry(logger), "   ")

		kubermaticConfig, helmValues, validationErrors := kubermaticStack.ValidateConfiguration(kubermaticConfig, helmValues, subLogger)
		if len(validationErrors) > 0 {
			logger.Error("⛔ The provided configuration files are invalid:")

			for _, e := range validationErrors {
				subLogger.Errorf("%v", e)
			}

			return errors.New("please review your configuration and try again")
		}

		logger.Info("✅ Provided configuration is valid.")

		// prepapre Kubernetes and Helm clients
		ctrlConfig, err := ctrlruntimeconfig.GetConfigWithContext(kubeContext)
		if err != nil {
			return fmt.Errorf("failed to get config: %v", err)
		}

		mgr, err := manager.New(ctrlConfig, manager.Options{})
		if err != nil {
			return fmt.Errorf("failed to construct mgr: %v", err)
		}

		// start the manager in its own goroutine
		go func() {
			if err := mgr.Start(wait.NeverStop); err != nil {
				logger.Fatalf("Failed to start Kubernetes client manager: %v", err)
			}
		}()

		appContext := context.Background()

		// wait for caches to be synced
		mgrSyncCtx, cancel := context.WithTimeout(appContext, 30*time.Second)
		defer cancel()
		if synced := mgr.GetCache().WaitForCacheSync(mgrSyncCtx.Done()); !synced {
			logger.Fatal("Timed out while waiting for Kubernetes client caches to synchronize.")
		}

		kubeClient := mgr.GetClient()

		if err := apiextensionsv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
			return fmt.Errorf("failed to add scheme: %v", err)
		}

		if err := kubermaticv1.AddToScheme(mgr.GetScheme()); err != nil {
			return fmt.Errorf("failed to add scheme: %v", err)
		}

		if err := operatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			return fmt.Errorf("failed to add scheme: %v", err)
		}

		if err := certmanagerv1alpha2.AddToScheme(mgr.GetScheme()); err != nil {
			return fmt.Errorf("failed to add scheme: %v", err)
		}

		opt := stack.DeployOptions{
			HelmClient:                 helmClient,
			KubeClient:                 kubeClient,
			HelmValues:                 helmValues,
			KubermaticConfiguration:    kubermaticConfig,
			RawKubermaticConfiguration: rawKubermaticConfig,
			Logger:                     subLogger,
			StorageClassProvider:       ctx.String(deployStorageClassFlag.Name),
			ForceHelmReleaseUpgrade:    ctx.Bool(deployForceFlag.Name),
			ChartsDirectory:            ctx.GlobalString(chartsDirectoryFlag.Name),
		}

		logger.Info("🧩 Deploying %s…", kubermaticStack.Name())

		if err := kubermaticStack.Deploy(appContext, opt); err != nil {
			return err
		}

		logger.Infof("🛬 Installation completed successfully. %s", greeting())

		return nil
	}))
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
