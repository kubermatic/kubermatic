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

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Scenario struct {
	Provider        string   `yaml:"provider"`
	OperatingSystem string   `yaml:"operatingSystem"`
	Flavors         []Flavor `yaml:"flavors"`
}

type Flavor struct {
	Name  string      `yaml:"name"`
	Value interface{} `yaml:"value"`
}

type Config struct {
	// Versions optionally defines the Kubernetes versions to use when running scenarios from this file.
	// If empty, the runner falls back to the versions provided via CLI flags.
	Versions  []string   `yaml:"versions,omitempty"`
	Scenarios []Scenario `yaml:"scenarios"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenario file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse scenario file: %w", err)
	}

	return config, nil
}
