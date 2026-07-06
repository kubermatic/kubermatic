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

package cni

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func TestDeprecatedCiliumVersionsAreAllowedButNotSupported(t *testing.T) {
	supported, err := GetSupportedCNIPluginVersions(kubermaticv1.CNIPluginTypeCilium)
	if err != nil {
		t.Fatalf("failed to get supported Cilium versions: %v", err)
	}

	allowed, err := GetAllowedCNIPluginVersions(kubermaticv1.CNIPluginTypeCilium)
	if err != nil {
		t.Fatalf("failed to get allowed Cilium versions: %v", err)
	}

	currentVersions := []string{"1.17.16", "1.18.10", "1.19.4"}
	for _, version := range currentVersions {
		if !supported.Has(version) {
			t.Errorf("expected Cilium %s to be supported", version)
		}
		if !allowed.Has(version) {
			t.Errorf("expected Cilium %s to be allowed", version)
		}
	}

	deprecatedVersions := []string{
		"1.15.16",
		"1.16.9",
		"1.17.7",
		"1.17.12",
		"1.17.14",
		"1.18.2",
		"1.18.6",
		"1.18.8",
	}
	for _, version := range deprecatedVersions {
		if supported.Has(version) {
			t.Errorf("expected Cilium %s to be deprecated, not supported", version)
		}
		if !allowed.Has(version) {
			t.Errorf("expected deprecated Cilium %s to remain allowed", version)
		}
	}

	defaultVersion := GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCilium)
	if !supported.Has(defaultVersion) {
		t.Errorf("expected default Cilium version %s to be supported", defaultVersion)
	}
}
