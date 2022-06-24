/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"

	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func GetImagesForHelmCharts(ctx context.Context, log logrus.FieldLogger, config *kubermaticv1.KubermaticConfiguration, helmClient helm.Client, chartsPath string, valuesFile string) ([]string, error) {
	if info, err := os.Stat(chartsPath); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%s is not a valid directory", chartsPath)
	}

	chartPaths, err := findHelmCharts(chartsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Helm charts: %w", err)
	}

	images := []string{}
	serializer := json.NewSerializer(&json.SimpleMetaFactory{}, scheme.Scheme, scheme.Scheme, false)

	for _, chartPath := range chartPaths {
		chartName := filepath.Base(chartPath)
		chartLog := log.WithFields(logrus.Fields{
			"path":  chartPath,
			"chart": chartName,
		})

		// do not render the Kubermatic chart again, if a Kubermatic configuration
		// is given; in this case, the operator is used and we determine the images
		// used via the static creators in Go code.
		if config != nil && chartName == "kubermatic" {
			chartLog.Debug("Skipping chart because KubermaticConfiguration was given")
			continue
		}

		chartLog.Debug("Fetching chart dependencies…")
		if err := helmClient.BuildChartDependencies(chartPath, nil); err != nil {
			return nil, fmt.Errorf("failed to download chart dependencies: %w", err)
		}

		chartLog.Debug("Rendering chart…")

		rendered, err := helmClient.RenderChart(mockNamespaceName, chartName, chartPath, valuesFile, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to render Helm chart %q: %w", chartName, err)
		}

		manifests, err := yamlutil.ParseMultipleDocuments(bytes.NewReader(rendered))
		if err != nil {
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}

		for _, manifest := range manifests {
			manifestImages, err := getImagesFromManifest(log, serializer, manifest.Raw)
			if err != nil {
				return nil, fmt.Errorf("failed to parse manifests: %w", err)
			}

			images = append(images, manifestImages...)
		}
	}

	return images, nil
}

// findHelmCharts walks the root directory and finds Chart.yaml files. It
// then returns the found directory paths (without the "/Chart.yaml" filename).
func findHelmCharts(root string) ([]string, error) {
	charts := []string{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if _, err := os.Stat(filepath.Join(path, "Chart.yaml")); err == nil {
				charts = append(charts, path)
				return filepath.SkipDir
			}
		}

		return nil
	})

	sort.Strings(charts)

	return charts, err
}
