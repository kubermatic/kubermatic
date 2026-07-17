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
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"

	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

// helmDepBuildMu serializes BuildChartDependencies across concurrent chart
// renders. helm writes registered repositories to a single shared file
// (~/.config/helm/repositories.yaml); concurrent `helm repo add` calls race on
// it and can corrupt or lose entries. The dependency download and template
// render themselves write to per-chart directories and stay concurrent.
var helmDepBuildMu sync.Mutex

func GetImagesForHelmCharts(ctx context.Context, log logrus.FieldLogger, config *kubermaticv1.KubermaticConfiguration, helmClient helm.Client, chartsPath string, valuesFile string, registryPrefix string, kubeVersion string, concurrency int) (*Collection, error) {
	if info, err := os.Stat(chartsPath); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%s is not a valid directory", chartsPath)
	}

	chartPaths, err := findHelmCharts(chartsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Helm charts: %w", err)
	}

	// findHelmCharts returns sorted paths, so the per-chart results stay
	// deterministic regardless of worker scheduling.
	type chartResult struct {
		chartPath string
		images    []string
	}

	if concurrency < 1 {
		concurrency = 1
	}

	jobs := make(chan string)
	results := make(chan chartResult)
	errCh := make(chan error, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	workerCount := concurrency
	if workerCount > len(chartPaths) {
		workerCount = len(chartPaths)
	}
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chartPath := range jobs {
				// a sibling worker hit an error; stop pulling new work
				if ctx.Err() != nil {
					return
				}

				chartImages, err := GetImagesForHelmChart(log, config, helmClient, chartPath, valuesFile, registryPrefix, kubeVersion, &helmDepBuildMu)
				if err != nil {
					select {
					case errCh <- fmt.Errorf("failed to get images for Helm chart %q: %w", filepath.Base(chartPath), err):
					default:
					}
					cancel()
					return
				}

				results <- chartResult{chartPath: chartPath, images: chartImages}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, chartPath := range chartPaths {
			if ctx.Err() != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case jobs <- chartPath:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	collected := make(map[string][]string, len(chartPaths))
	for res := range results {
		collected[res.chartPath] = res.images
	}

	select {
	case err := <-errCh:
		return nil, err
	default:
	}

	collection := NewCollection()
	// iterate in the original sorted order for deterministic output
	for _, chartPath := range chartPaths {
		collection.InsertAll(collected[chartPath], RefTypeImage, "chart:"+filepath.Base(chartPath))
	}

	return collection, nil
}

func GetImagesForHelmChart(log logrus.FieldLogger, config *kubermaticv1.KubermaticConfiguration, helmClient helm.Client, chartPath string, valuesFile string, registryPrefix string, kubeVersion string, depBuildMu *sync.Mutex) ([]string, error) {
	images := []string{}
	serializer := json.NewSerializer(&json.SimpleMetaFactory{}, scheme.Scheme, scheme.Scheme, false)

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
		return nil, nil
	}

	// fetch dependencies only for charts in folders.
	// Chart packages (i.e. the tgz) should already contain them.
	chartFI, err := os.Stat(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat chart path: %w", err)
	}
	if chartFI.IsDir() {
		chartLog.Debug("Fetching chart dependencies…")
		// BuildChartDependencies runs `helm repo add` which mutates the shared
		// repositories.yaml; serialize it across concurrent renders when a mutex
		// is provided. The dependency download and template render that follow
		// are per-chart and stay concurrent.
		if depBuildMu != nil {
			depBuildMu.Lock()
		}
		err := helmClient.BuildChartDependencies(chartPath, nil)
		if depBuildMu != nil {
			depBuildMu.Unlock()
		}
		if err != nil {
			return nil, fmt.Errorf("failed to download chart dependencies: %w", err)
		}
	}

	chartLog.Debug("Rendering chart…")

	// we need to set this flag for RenderChart to ensure compatibility with charts that specify a kubeVersion field (e.g. mla/consul)
	versionFlag := []string{}
	if kubeVersion != "" {
		versionFlag = []string{"--kube-version", kubeVersion}
	}

	rendered, err := helmClient.RenderChart(mockNamespaceName, chartName, chartPath, valuesFile, nil, versionFlag)
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

	if registryPrefix != "" {
		var filteredImages []string
		for _, image := range images {
			if strings.HasPrefix(image, registryPrefix) {
				filteredImages = append(filteredImages, image)
			}
		}

		images = filteredImages
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
