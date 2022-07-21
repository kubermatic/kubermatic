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

package kubermatic

import (
	"k8c.io/kubermatic/v2/pkg/util/edition"
)

// These variables get fed by ldflags during compilation.
var (
	// gitVersion is a magic variable containing the git commit identifier
	// (usually the output of `git describe`, i.e. not necessarily a
	// static tag name); for a tagged KKP release, this value is identical
	// to kubermaticDockerTag, for untagged builds this is the `git describe`
	// output.
	// Importantly, this value will only ever go up, even for untagged builds,
	// but is not monotone (gaps can occur, this can go from v2.20.0-1234-d6aef3
	// to v2.34.0-912-dd79178e to v3.0.1).
	// Also this value does not necessarily reflect the current release branch,
	// as releases are taggedo on the release branch and on those tags are not
	// visible from the master branch.
	gitVersion string

	// kubermaticDockerTag is a magic variable containing the tag / git commit hash
	// of the kubermatic Docker image to deploy. For tagged releases this is
	// identical to gitVersion, but for nightly builds, this is identical to the
	// gitVersion.
	kubermaticDockerTag string

	// uiDockerTag is a magic variable containing the tag / git commit hash
	// of the dashboard Docker image to deploy.
	uiDockerTag string
)

type Versions struct {
	KubermaticCommit  string
	Kubermatic        string
	UI                string
	VPA               string
	Envoy             string
	KubermaticEdition edition.Type
}

func NewDefaultVersions() Versions {
	return Versions{
		KubermaticCommit:  gitVersion,
		Kubermatic:        kubermaticDockerTag,
		UI:                uiDockerTag,
		VPA:               "0.11.0",
		Envoy:             "v1.17.1",
		KubermaticEdition: edition.KubermaticEdition,
	}
}

func NewFakeVersions() Versions {
	return Versions{
		KubermaticCommit:  "v0.0.0-420-test",
		Kubermatic:        "v0.0.0-test",
		UI:                "v1.1.1-test",
		VPA:               "0.5.0",
		Envoy:             "v1.16.0",
		KubermaticEdition: edition.KubermaticEdition,
	}
}
