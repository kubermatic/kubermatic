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

package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v2"
)

type Release struct {
	Name      string          `json:"name"`
	Namespace string          `json:"namespace"`
	Chart     string          `json:"chart"`
	Revision  string          `json:"revision"`
	Version   *semver.Version `json:"-"`
	// AppVersion is not a semver, for example Minio has date-based versions.
	AppVersion string        `json:"app_version"`
	Status     ReleaseStatus `json:"status"`
}

func (r *Release) Clone() Release {
	copy := *r
	copy.Version = semver.MustParse(r.Version.Original())

	return copy
}

type Chart struct {
	Name       string          `yaml:"name"`
	Version    *semver.Version `yaml:"-"`
	VersionRaw string          `yaml:"version"`
	// AppVersion is not a semver, for example Minio has date-based versions.
	AppVersion   string `yaml:"appVersion"`
	Directory    string
	Dependencies []Dependency `yaml:"dependencies,omitempty"`
}

func (c *Chart) Clone() Chart {
	copy := *c
	copy.Version = semver.MustParse(c.Version.Original())

	return copy
}

func LoadChart(directory string) (*Chart, error) {
	f, err := os.Open(filepath.Join(directory, "Chart.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to open Chart.yaml: %w", err)
	}
	defer f.Close()

	chart := &Chart{}
	if err := yaml.NewDecoder(f).Decode(chart); err != nil {
		return nil, fmt.Errorf("failed to read Chart.yaml: %w", err)
	}

	version, err := semver.NewVersion(chart.VersionRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %q: %w", chart.VersionRaw, err)
	}

	chart.Version = version
	chart.Directory = directory

	return chart, nil
}

type Dependency struct {
	Name         string        `yaml:"name"`
	Version      string        `yaml:"version,omitempty"`
	Repository   string        `yaml:"repository"`
	Condition    string        `yaml:"condition,omitempty"`
	Tags         []string      `yaml:"tags,omitempty"`
	Enabled      bool          `yaml:"enabled,omitempty"`
	ImportValues []interface{} `json:"import-values,omitempty"`
	Alias        string        `yaml:"alias,omitempty"`
}
