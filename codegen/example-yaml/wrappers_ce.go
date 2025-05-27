//go:build !ee

/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func createExampleSeed(config *kubermaticv1.KubermaticConfiguration) *kubermaticv1.Seed {
	// Make substitutions for CE version
	seed := createBaseExampleSeed(config)
	seed.Spec.KubeLB = nil
	if dc, ok := seed.Spec.Datacenters[sampledc]; ok {
		dc.Spec.KubeLB = nil
		seed.Spec.Datacenters[sampledc] = dc
	}
	return seed
}
