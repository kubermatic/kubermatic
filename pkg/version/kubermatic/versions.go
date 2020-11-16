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

// UIDOCKERTAG is a magic variable containing the tag / git commit hash
// of the dashboard Docker image to deploy. It gets fed by the
// Makefile as an ldflag.
var UIDOCKERTAG string

// KUBERMATICDOCKERTAG is a magic variable containing the tag / git commit hash
// of the kubermatic Docker image to deploy. It gets fed by the
// Makefile as an ldflag.
var KUBERMATICDOCKERTAG string

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
		KubermaticCommit:  "tbd",
		Kubermatic:        KUBERMATICDOCKERTAG,
		UI:                UIDOCKERTAG,
		VPA:               "0.5.0",
		Envoy:             "v1.16.0",
		KubermaticEdition: edition.KubermaticEdition,
	}
}
