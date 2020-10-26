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

import "github.com/Masterminds/semver"

// Client describes the operations that the Helm client is providing to
// the installer. This is the minimum set of operations required to
// perform a Kubermatic installation.
type Client interface {
	Version() (*semver.Version, error)
	InstallChart(namespace string, releaseName string, chartDirectory string, valuesFile string, values map[string]string, flags []string) error
	GetRelease(namespace string, name string) (*Release, error)
	ListReleases(namespace string) ([]Release, error)
	UninstallRelease(namespace string, name string) error
	RenderChart(namespace string, releaseName string, chartDirectory string, valuesFile string, values map[string]string) ([]byte, error)
}
