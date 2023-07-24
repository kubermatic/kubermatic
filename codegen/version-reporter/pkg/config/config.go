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
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Products []SoftwareProduct `yaml:"products" json:"products"`
}

type SoftwareProduct struct {
	Name        string             `yaml:"name" json:"name"`
	SourceURL   string             `yaml:"source" json:"source"`
	Occurrences []VersionReference `yaml:"occurrences" json:"occurrences"`
}

// Unversioned is the key in the VersionReference.Versions
// for all reference types that only produce a single version,
// instead of depending on the user-cluster version.
const Unversioned = "*"

type VersionReference struct {
	GoConstant *GoConstantReference `yaml:"goConstant,omitempty" json:"goConstant,omitempty"`
	GoFunction *GoFunctionReference `yaml:"goFunction,omitempty" json:"goFunction,omitempty"`
	HelmChart  *HelmChartReference  `yaml:"helmChart,omitempty" json:"helmChart,omitempty"`
	YAMLFile   *YAMLFileReference   `yaml:"yamlFile,omitempty" json:"yamlFile,omitempty"`

	// Versions is filled during runtime (JSON tag exists for output purposes)
	Versions map[string]string `json:"versions"`
}

func (r VersionReference) String() string {
	switch {
	case r.GoConstant != nil:
		return r.GoConstant.String()
	case r.GoFunction != nil:
		return r.GoFunction.String()
	case r.HelmChart != nil:
		return r.HelmChart.String()
	case r.YAMLFile != nil:
		return r.YAMLFile.String()
	default:
		return fmt.Sprintf("%#v", r)
	}
}

func (r VersionReference) TypeName() string {
	switch {
	case r.GoConstant != nil:
		return r.GoConstant.TypeName()
	case r.GoFunction != nil:
		return r.GoFunction.TypeName()
	case r.HelmChart != nil:
		return r.HelmChart.TypeName()
	case r.YAMLFile != nil:
		return r.YAMLFile.TypeName()
	default:
		return fmt.Sprintf("%T", r)
	}
}

type GoConstantReference struct {
	Package  string `yaml:"package" json:"package"`
	Constant string `yaml:"constant" json:"constant"`
}

func (r GoConstantReference) TypeName() string {
	return "Go const"
}

func (r GoConstantReference) String() string {
	return fmt.Sprintf("%s.%s", r.Package, r.Constant)
}

type GoFunctionReference struct {
	Function string `yaml:"function" json:"function"`
}

func (r GoFunctionReference) String() string {
	return r.Function
}

func (r GoFunctionReference) TypeName() string {
	return "Go func"
}

type HelmChartReference struct {
	Directory string `yaml:"directory" json:"directory"`
	ValuePath string `yaml:"valuePath,omitempty" json:"valuePath,omitempty"`
}

func (r HelmChartReference) String() string {
	if r.ValuePath != "" {
		return fmt.Sprintf("%s @ %s", r.Directory, r.ValuePath)
	}
	return r.Directory
}

func (r HelmChartReference) TypeName() string {
	return "Helm chart"
}

type YAMLFileReference struct {
	File      string `yaml:"file" json:"file"`
	ValuePath string `yaml:"valuePath" json:"valuePath"`
}

func (r YAMLFileReference) String() string {
	return fmt.Sprintf("%s @ %s", r.File, r.ValuePath)
}

func (r YAMLFileReference) TypeName() string {
	return "YAML file"
}

func Load(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	cfg := &Config{}
	if err := decoder.Decode(cfg); err != nil {
		return nil, err
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration is invalid: %w", err)
	}

	return cfg, nil
}

func (c *Config) Sort() {
	sort.Slice(c.Products, func(i, j int) bool {
		return strings.ToLower(c.Products[i].Name) < strings.ToLower(c.Products[j].Name)
	})

	for idx, product := range c.Products {
		sort.Slice(product.Occurrences, func(i, j int) bool {
			a := product.Occurrences[i]
			b := product.Occurrences[j]

			if a.TypeName() != b.TypeName() {
				return a.TypeName() < b.TypeName()
			}

			return a.String() < b.String()
		})

		c.Products[idx] = product
	}
}

func validateConfig(cfg *Config) error {
	for idx, product := range cfg.Products {
		if product.Name == "" {
			return fmt.Errorf("product #%d has no name", idx)
		}

		if product.SourceURL == "" {
			return fmt.Errorf("product %q has no source URL", product.Name)
		}

		for jdx, occ := range product.Occurrences {
			switch {
			case occ.GoConstant != nil:
				continue
			case occ.GoFunction != nil:
				continue
			case occ.HelmChart != nil:
				continue
			case occ.YAMLFile != nil:
				continue
			default:
				return fmt.Errorf("occurrence #%d of %q has no valid locator defined", jdx, product.Name)
			}
		}
	}

	return nil
}
