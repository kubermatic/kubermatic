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

package reader

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func ReadHelmChartVersion(chart string, valuePath string) (string, error) {
	if valuePath == "" {
		return readHelmChartAppVersion(chart)
	}

	return readHelmChartValue(chart, valuePath)
}

type helmChart struct {
	AppVersion string `yaml:"appVersion"`
}

func readHelmChartAppVersion(dir string) (string, error) {
	yamlFile := filepath.Join(dir, "Chart.yaml")

	f, err := os.Open(yamlFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	chart := helmChart{}
	if err := yaml.NewDecoder(f).Decode(&chart); err != nil {
		return "", err
	}

	return chart.AppVersion, nil
}

func readHelmChartValue(dir string, valuePath string) (string, error) {
	return ReadYAMLVersion(filepath.Join(dir, "values.yaml"), valuePath)
}
