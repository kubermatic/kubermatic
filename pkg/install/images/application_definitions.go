/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package images

import (
	"context"
	"fmt"
	"iter"
	"os"
	"path"
	"time"

	"github.com/sirupsen/logrus"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applicationdefinitions"
	"k8c.io/kubermatic/v2/pkg/applications/providers"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"sigs.k8s.io/yaml"
)

func GetImagesFromSystemApplicationDefinitions(
	logger logrus.FieldLogger,
	config *kubermaticv1.KubermaticConfiguration,
	helmClient helm.Client,
	helmTimeout time.Duration,
	registryPrefix string,
) ([]string, error) {
	var images []string

	for sysApp, err := range SystemAppsHelmCharts(config, logger, helmClient, helmTimeout, registryPrefix) {
		if err != nil {
			return nil, err
		}

		images = append(images, sysApp.WorkloadImages...)
	}

	return images, nil
}

type SystemAppsHelmChart struct {
	ChartArchive   string
	WorkloadImages []string
	appskubermaticv1.ApplicationVersion
}

func SystemAppsHelmCharts(
	config *kubermaticv1.KubermaticConfiguration,
	logger logrus.FieldLogger,
	helmClient helm.Client,
	helmTimeout time.Duration,
	registryPrefix string,
) iter.Seq2[*SystemAppsHelmChart, error] {
	// If system applications are disabled we don't need to do anything.
	if config.Spec.SystemApplications.Disable {
		logger.Debug("System applications are disabled, skipping deployment of system application definitions.")
		return nil
	}

	log := kubermaticlog.NewDefault().Sugar()
	sysAppDefReconcilers, err := applicationdefinitions.SystemApplicationDefinitionReconcilerFactories(log, config)
	if err != nil {
		return func(yield func(*SystemAppsHelmChart, error) bool) {
			yield(nil, fmt.Errorf("failed to get system application definition reconciler factories: %w", err))
		}
	}

	return func(yield func(*SystemAppsHelmChart, error) bool) {
		for _, createFunc := range sysAppDefReconcilers {
			appName, creator := createFunc()
			appDef, err := creator(&appskubermaticv1.ApplicationDefinition{})
			if err != nil {
				yield(nil, err)

				return
			}
			appLog := logger.WithFields(logrus.Fields{"application-name": appName})

			if appDef.Spec.Method != appskubermaticv1.HelmTemplateMethod {
				// Only Helm ApplicationDefinitions are supported at the moment
				appLog.Debugf("Skipping the ApplicationDefinition as the method '%s' is not supported yet", appDef.Spec.Method)
				break
			}
			appLog.Info("Retrieving images…")

			tmpDir, err := os.MkdirTemp("", "helm-charts")
			if err != nil {
				yield(nil, fmt.Errorf("failed to create temp dir: %w", err))

				return
			}
			defer func() {
				if err := os.RemoveAll(tmpDir); err != nil {
					appLog.Fatalf("Failed to remove temp dir: %v", err)
				}
			}()

			// if DefaultValues is provided, use it as values file
			valuesFile := ""
			values, err := appDef.Spec.GetDefaultValues()
			if err != nil {
				yield(nil, err)
				return
			}

			if appName == kubermaticv1.CNIPluginTypeCilium.String() {
				// For Cilium, we'll use the default values from the Helm chart but will mutate the values to ensure that additional images are also mirrored.
				defaultValues, err := appDef.Spec.GetParsedDefaultValues()
				if err != nil {
					yield(nil, fmt.Errorf("failed to unmarshal CNI values: %w", err))
					return
				}
				// Enable `cilium-envoy` to ensure that the additional images are also mirrored.
				if envoy, ok := defaultValues["envoy"].(map[string]any); ok {
					envoy["enabled"] = true
				} else {
					defaultValues["envoy"] = map[string]any{
						"enabled": true,
					}
				}

				values, err = yaml.Marshal(defaultValues)
				if err != nil {
					yield(nil, fmt.Errorf("failed to marshal CNI values: %w", err))
					return
				}
			}

			if values != nil {
				valuesFile = path.Join(tmpDir, "values.yaml")
				err = os.WriteFile(valuesFile, values, 0o644)
				if err != nil {
					yield(nil, fmt.Errorf("failed to create values file: %w", err))
					return
				}
			}

			for _, appVer := range appDef.Spec.Versions {
				appVerLog := appLog.WithField("application-version", appVer.Version)
				appVerLog.Debug("Downloading Helm chart…")
				// pull the chart
				chartPath, err := downloadAppSourceChart(&appVer.Template.Source, tmpDir, helmTimeout)
				if err != nil {
					yield(nil, fmt.Errorf("failed to pull app chart: %w", err))

					return
				}

				// get images
				chartImages, err := GetImagesForHelmChart(appVerLog, nil, helmClient, chartPath, valuesFile, registryPrefix, "") // since we don't have the version constraints in AppDefs yet, we can leave kubeVersion parameter empty
				if err != nil {
					yield(nil, fmt.Errorf("failed to get images for chart: %w", err))

					return
				}

				sysChart := SystemAppsHelmChart{
					ChartArchive:       chartPath,
					WorkloadImages:     chartImages,
					ApplicationVersion: appVer,
				}

				if !yield(&sysChart, nil) {
					return
				}
			}
		}
	}
}

func downloadAppSourceChart(appSource *appskubermaticv1.ApplicationSource, directory string, timeout time.Duration) (chartPath string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sp, err := providers.NewSourceProvider(ctx, kubermaticlog.NewDefault().Sugar(), nil, "", directory, appSource, "")
	if err != nil {
		return "", fmt.Errorf("failed to create app source provider: %w", err)
	}

	chartPath, err = sp.DownloadSource(directory)
	if err != nil {
		return "", fmt.Errorf("failed to download app source: %w", err)
	}

	return chartPath, nil
}
