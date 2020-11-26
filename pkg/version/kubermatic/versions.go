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
	// gitHash is a magic variable containing the git commit hash
	// of the current (as in currently executing) kubermatic api.
	gitHash string

	// kubermaticDockerTag is a magic variable containing the tag / git commit hash
	// of the kubermatic Docker image to deploy.
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
		KubermaticCommit:  gitHash,
		Kubermatic:        kubermaticDockerTag,
		UI:                uiDockerTag,
		VPA:               "0.5.0",
		Envoy:             "v1.16.0",
		KubermaticEdition: edition.KubermaticEdition,
	}
}

func NewFakeVersions() Versions {
	return Versions{
		KubermaticCommit:  "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		Kubermatic:        "v0.0.0-test",
		UI:                "v1.1.1-test",
		VPA:               "0.5.0",
		Envoy:             "v1.16.0",
		KubermaticEdition: edition.KubermaticEdition,
	}
}
