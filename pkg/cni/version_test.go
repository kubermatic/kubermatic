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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
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

	currentVersions := []string{"1.17.16", "1.18.10", "1.18.13", "1.19.4", "1.19.7"}
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

func TestDefaultCNIVersionsAreSupported(t *testing.T) {
	for cniType, defaultVersion := range defaultCNIPluginVersion {
		if !supportedCNIPluginVersions[cniType].Has(defaultVersion) {
			t.Errorf("default %s version %s is not a supported version", cniType, defaultVersion)
		}
	}
}

func TestSupportedAndDeprecatedCNIVersionsAreDisjoint(t *testing.T) {
	for cniType, supported := range supportedCNIPluginVersions {
		if both := supported.Intersection(deprecatedCNIPluginVersions[cniType]); both.Len() > 0 {
			t.Errorf("%s versions %v are both supported and deprecated", cniType, sets.List(both))
		}
	}
}

// Canal is deployed as an addon with one manifest per version. A manifest is only
// rendered if its version guard matches the cluster's CNI version, so a missing
// file or a wrong guard leaves a cluster without any CNI.
func TestCanalAddonManifestsMatchAllowedVersions(t *testing.T) {
	const addonDir = "../../addons/canal"

	allowed, err := GetAllowedCNIPluginVersions(kubermaticv1.CNIPluginTypeCanal)
	if err != nil {
		t.Fatalf("failed to get allowed Canal versions: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(addonDir, "canal_v*.yaml"))
	if err != nil {
		t.Fatalf("failed to list Canal addon manifests: %v", err)
	}

	available := sets.New[string]()
	for _, file := range files {
		version := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(file), "canal_"), ".yaml")
		available.Insert(version)

		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		guard := fmt.Sprintf(`{{ if eq .Cluster.CNIPlugin.Version %q }}`, version)
		if !strings.Contains(string(content), guard) {
			t.Errorf("%s does not contain the version guard %s", file, guard)
		}
	}

	if missing := allowed.Difference(available); missing.Len() > 0 {
		t.Errorf("allowed Canal versions %v have no manifest in %s", sets.List(missing), addonDir)
	}
	if unknown := available.Difference(allowed); unknown.Len() > 0 {
		t.Errorf("Canal manifests for versions %v exist in %s, but these versions are not allowed", sets.List(unknown), addonDir)
	}
}
