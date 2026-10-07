/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package jig

import (
	"testing"

	"go.uber.org/zap"

	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"
)

func TestResolveMinorVersion(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []string{
		"v1.37.0",
		"1.37.0",
		"v1.99",
		"garbage",
		"",
	}

	for _, input := range testCases {
		if output := resolveMinorVersion(log, input); output != input {
			t.Errorf("expected %q to stay unchanged, got %q", input, output)
		}
	}

	// resolving the default version's minor must return the newest supported
	// patch release of that minor
	def := defaulting.DefaultKubernetesVersioning.Default
	resolved, err := semver.NewSemver(resolveMinorVersion(log, "v"+def.MajorMinor()))
	if err != nil {
		t.Fatalf("resolved version is not a valid semver: %v", err)
	}

	for i := range defaulting.DefaultKubernetesVersioning.Versions {
		sv := &defaulting.DefaultKubernetesVersioning.Versions[i]
		if sv.MajorMinor() == resolved.MajorMinor() && sv.GreaterThan(resolved) {
			t.Errorf("resolved to %q, but %q is a newer supported patch release", resolved.String(), sv.String())
		}
	}
}
