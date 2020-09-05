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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack/kubermatic"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/edition"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func DeployCommand(logger *logrus.Logger) cli.Command {
	return cli.Command{
		Name:   "deploy",
		Usage:  "Installs or upgrades the current installation to the installer's built-in version",
		Action: DeployAction(logger),
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "force",
				Usage: "Perform Helm upgrades even when the release is up-to-date",
			},
			cli.StringFlag{
				Name:   "config",
				Usage:  "Full path to the KubermaticConfiguration YAML file",
				EnvVar: "CONFIG_YAML",
			},
			cli.StringFlag{
				Name:   "helm-values",
				Usage:  "Full path to the Helm values.yaml used for customizing all charts",
				EnvVar: "VALUES_YAML",
			},
			cli.StringFlag{
				Name:   "kubeconfig",
				Usage:  "Full path to where a kubeconfig with cluster-admin permissions for the target cluster",
				EnvVar: "KUBECONFIG",
			},
			cli.StringFlag{
				Name:   "kube-context",
				Usage:  "Context to use from the given kubeconfig",
				EnvVar: "KUBE_CONTEXT",
			},
			cli.DurationFlag{
				Name:  "helm-timeout",
				Usage: "Time to wait for Helm operations to finish",
				Value: 5 * time.Minute,
			},
			cli.StringFlag{
				Name:   "helm-binary",
				Usage:  "Full path to the Helm 3 binary to use",
				Value:  "helm",
				EnvVar: "HELM_BINARY",
			},
		},
	}
}

func DeployAction(logger *logrus.Logger) cli.ActionFunc {
	return handleErrors(logger, setupLogger(logger, func(ctx *cli.Context) error {
		v := common.NewDefaultVersions()

		fields := logrus.Fields{
			"version": v.Kubermatic,
			"edition": edition.KubermaticEdition,
		}
		if ctx.GlobalBool("verbose") {
			fields["git"] = resources.KUBERMATICCOMMIT
		}

		logger.WithFields(fields).Info("🛫 Initializing installer…")

		subLogger := log.Prefix(logrus.NewEntry(logger), "   ")

		// load config files
		kubermaticConfig, rawKubermaticConfig, err := loadKubermaticConfiguration(ctx.String("config"))
		if err != nil {
			return fmt.Errorf("failed to load KubermaticConfiguration: %v", err)
		}

		helmValues, err := loadHelmValues(ctx.String("helm-values"))
		if err != nil {
			return fmt.Errorf("failed to load Helm values: %v", err)
		}

		// validate the configuration
		logger.Info("🚦 Validating the provided configuration…")

		kubermaticConfig, helmValues, validationErrors := kubermatic.ValidateConfiguration(kubermaticConfig, helmValues, subLogger)
		if len(validationErrors) > 0 {
			logger.Error("⛔ The provided configuration files are invalid:")

			for _, e := range validationErrors {
				subLogger.Errorf("%v", e)
			}

			return errors.New("please review your configuration and try again")
		}

		logger.Info("✅ Provided configuration is valid.")

		// prepapre Kubernetes and Helm clients
		kubeconfig := ctx.String("kubeconfig")
		if len(kubeconfig) == 0 {
			return errors.New("no kubeconfig (--kubeconfig or $KUBECONFIG) given")
		}

		kubeContext := ctx.String("kube-context")
		helmTimeout := ctx.Duration("helm-timeout")
		helmBinary := ctx.String("helm-binary")

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

		if err := operatorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
			return fmt.Errorf("failed to add scheme: %v", err)
		}

		if err := certmanagerv1alpha2.AddToScheme(mgr.GetScheme()); err != nil {
			return fmt.Errorf("failed to add scheme: %v", err)
		}

		helmClient, err := helm.NewCLI(helmBinary, kubeconfig, kubeContext, helmTimeout, logger)
		if err != nil {
			return fmt.Errorf("failed to create Helm client: %v", err)
		}

		logger.Info("🧩 Deploying kubermatic stack…")
		opt := kubermatic.Options{
			HelmValues:                 helmValues,
			KubermaticConfiguration:    kubermaticConfig,
			RawKubermaticConfiguration: rawKubermaticConfig,
			ForceHelmReleaseUpgrade:    ctx.Bool("force"),
			ChartsDirectory:            ctx.GlobalString("charts-directory"),
		}

		if err := kubermatic.Deploy(appContext, subLogger, kubeClient, helmClient, opt); err != nil {
			return err
		}

		logger.Infof("🛬 Installation completed successfully. %s", greeting())

		return nil
	}))
}

func loadKubermaticConfiguration(filename string) (*operatorv1alpha1.KubermaticConfiguration, *unstructured.Unstructured, error) {
	if filename == "" {
		return nil, nil, errors.New("no file specified via --config flag")
	}

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	raw := &unstructured.Unstructured{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 1024).Decode(raw); err != nil {
		return nil, nil, fmt.Errorf("failed to decode %s: %v", filename, err)
	}

	config := &operatorv1alpha1.KubermaticConfiguration{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 1024).Decode(config); err != nil {
		return nil, raw, fmt.Errorf("failed to decode %s: %v", filename, err)
	}

	return config, raw, nil
}

func loadHelmValues(filename string) (*yamled.Document, error) {
	if filename == "" {
		return nil, errors.New("no file specified via --helm-values flag")
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	values, err := yamled.Load(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %v", filename, err)
	}

	return values, nil
}

func greeting() string {
	rand.Seed(time.Now().UnixNano())

	greetings := []string{
		"Have a nice day!",
		"Time for a break, maybe? ☺",
		"✌",
		"Thank you for using Kubermatic ❤",
	}

	return greetings[rand.Intn(len(greetings))]
}
