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

import "k8c.io/kubermatic/v2/pkg/util/edition"

// These variables get fed by ldflags during compilation.
var (
	// gitVersion is a magic variable containing the git commit identifier
	// (usually the output of `git describe`, i.e. not necessarily a
	// static tag name); for a tagged KKP release, this value is identical
	// to kubermaticContainerTag, for untagged builds this is the `git describe`
	// output.
	// Importantly, this value will only ever go up, even for untagged builds,
	// but is not monotone (gaps can occur, this can go from v2.20.0-1234-d6aef3
	// to v2.34.0-912-dd79178e to v3.0.1).
	// Also this value does not necessarily reflect the current release branch,
	// as releases are tagged on the release branch and on those tags are not
	// visible from the main branch.
	gitVersion string

	// kubermaticContainerTag is a magic variable containing the tag of the
	// kubermatic container image to deploy. For tagged releases this is
	// identical to gitVersion, but for nightly builds, this is identical to the
	// git HEAD hash that was built.
	kubermaticContainerTag string

	// uiContainerTag is a magic variable containing the tag of the dashboard
	// container image to deploy.
	uiContainerTag string
)

type Versions struct {
	GitVersion             string
	KubermaticContainerTag string
	UIContainerTag         string
	KubermaticEdition      edition.Type
}

func GetVersions() Versions {
	return Versions{
		GitVersion:             gitVersion,
		KubermaticContainerTag: kubermaticContainerTag,
		UIContainerTag:         uiContainerTag,
		KubermaticEdition:      edition.KubermaticEdition,
	}
}

func GetFakeVersions() Versions {
	return Versions{
		GitVersion:             "v0.0.0-420-test",
		KubermaticContainerTag: "v0.0.0-test",
		UIContainerTag:         "v1.1.1-test",
		KubermaticEdition:      edition.KubermaticEdition,
	}
}
