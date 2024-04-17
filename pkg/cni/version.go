/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

// CanalCNILastUnspecifiedVersion is the last Canal CNI version applied in KKP user-clusters
// started in KKP before v2.18 release. If cluster.spec.cniPlugin is not set, it means Canal of this version.
const CanalCNILastUnspecifiedVersion = "v3.8"

var (
	defaultCNIPluginVersion = map[kubermaticv1.CNIPluginType]string{
		kubermaticv1.CNIPluginTypeCanal:  "v3.27",
		kubermaticv1.CNIPluginTypeCilium: "1.13.8",
	}
)

var (
	// supportedCNIPlugins contains a list of all currently supported CNI Plugin types.
	supportedCNIPlugins = sets.New(
		kubermaticv1.CNIPluginTypeCanal.String(),
		kubermaticv1.CNIPluginTypeCilium.String(),
		kubermaticv1.CNIPluginTypeNone.String(),
	)
	// supportedCNIPluginVersions contains a list of all currently supported CNI versions for each CNI type.
	// Only supported versions are available for selection in KKP UI.
	supportedCNIPluginVersions = map[kubermaticv1.CNIPluginType]sets.Set[string]{
		kubermaticv1.CNIPluginTypeCanal: sets.New("v3.23", "v3.24", "v3.25", "v3.27"),
		kubermaticv1.CNIPluginTypeCilium: sets.New(
			"v1.11",
			"v1.12",
			// NOTE: as of 1.13.0, we moved to Application infra for Cilium CNI management and started using real smever
			// See pkg/cni/cilium docs for details on introducing a new version.
			"1.13.8",
		),
		kubermaticv1.CNIPluginTypeNone: sets.New(""),
	}
	// deprecatedCNIPluginVersions contains a list of deprecated CNI versions for each CNI type.
	// Deprecated versions are not available for selection in KKP UI, but are still accepted
	// by the validation webhook for backward compatibility.
	deprecatedCNIPluginVersions = map[kubermaticv1.CNIPluginType]sets.Set[string]{
		kubermaticv1.CNIPluginTypeCanal: sets.New("v3.8", "v3.19", "v3.20", "v3.21", "v3.22"),
		kubermaticv1.CNIPluginTypeCilium: sets.New(
			"1.13.0", // CVE-2023-34242
			"1.13.3", // CVE-2023-34242
			"1.13.4", // GHSA-pvgm-7jpg-pw5g, GHSA-69vr-g55c-v2v4, GHSA-mc6h-6j9x-v3gq, GHSA-7mhv-gr67-hq55
			"1.13.6", // CVE-2023-39347, CVE-2023-41333 (Moderate Severity), CVE-2023-41332 (Low Severity)
			"1.13.7", // CVE-2023-44487 (High Severity)
		),
	}
)

// AllowedCNIVersionTransition defines conditions for an allowed CNI version transition.
// If one of the versions is not specified, it means that it is not checked (always satisfied).
type AllowedCNIVersionTransition struct {
	K8sVersion    string
	OldCNIVersion string
	NewCNIVersion string
}

// allowedCNIVersionTransitions contains a map of allowed CNI version transitions for each CNI type.
// Apart from these, one minor version change is allowed for each CNI.
var allowedCNIVersionTransitions = map[kubermaticv1.CNIPluginType][]AllowedCNIVersionTransition{
	kubermaticv1.CNIPluginTypeCanal: {
		// allow upgrade from Canal v3.8 to any newer Canal version
		{
			K8sVersion:    "", // any
			OldCNIVersion: "= 3.8",
			NewCNIVersion: "> 3.8",
		},
		// allow upgrade to Canal v3.22 necessary for k8s >= v1.23
		{
			K8sVersion:    ">= 1.23",
			OldCNIVersion: "< 3.22",
			NewCNIVersion: "= 3.22",
		},
	},
}

// GetSupportedCNIPlugins returns currently supported CNI Plugin types.
func GetSupportedCNIPlugins() sets.Set[string] {
	return supportedCNIPlugins
}

// GetSupportedCNIPluginVersions returns currently supported CNI versions for a CNI type.
func GetSupportedCNIPluginVersions(cniPluginType kubermaticv1.CNIPluginType) (sets.Set[string], error) {
	if !supportedCNIPlugins.Has(cniPluginType.String()) {
		return sets.New[string](), fmt.Errorf("CNI Plugin type %q not supported. Supported types %s", cniPluginType, sets.List(supportedCNIPlugins))
	}

	versions, ok := supportedCNIPluginVersions[cniPluginType]
	// this means we messed up, should not happen as we support the plugin above
	if !ok {
		return sets.New[string](), fmt.Errorf("no versions available for CNI plugin %q", cniPluginType)
	}

	return versions, nil
}

// GetAllowedCNIPluginVersions returns all allowed CNI versions for a CNI type (supported + deprecated).
func GetAllowedCNIPluginVersions(cniPluginType kubermaticv1.CNIPluginType) (sets.Set[string], error) {
	supported, err := GetSupportedCNIPluginVersions(cniPluginType)
	if err != nil {
		return sets.New[string](), err
	}
	allowed := sets.New(sets.List(supported)...)
	if deprecated, ok := deprecatedCNIPluginVersions[cniPluginType]; ok {
		allowed.Insert(sets.List(deprecated)...)
	}
	return allowed, nil
}

// GetDefaultCNIPluginVersion returns the default CNI versions for a CNI type, empty string if no default version set.
func GetDefaultCNIPluginVersion(cniPluginType kubermaticv1.CNIPluginType) string {
	return defaultCNIPluginVersion[cniPluginType]
}

// IsSupportedCNIPluginTypeAndVersion returns true if the given CNI plugin is of supported type and version.
func IsSupportedCNIPluginTypeAndVersion(cni *kubermaticv1.CNIPluginSettings) bool {
	if cni == nil {
		return false
	}
	versions, ok := supportedCNIPluginVersions[cni.Type]
	if !ok {
		return false
	}
	if versions.Has(cni.Version) {
		return true
	}
	return false
}

// GetAllowedCNIVersionTransitions returns a list of allowed CNI version transitions for the given CNI type.
// Apart from these, one minor version change is allowed for each CNI.
func GetAllowedCNIVersionTransitions(cniPluginType kubermaticv1.CNIPluginType) []AllowedCNIVersionTransition {
	return allowedCNIVersionTransitions[cniPluginType]
}

// IsManagedByAppInfra returns true if the given CNI type and version is managed by KKP Applications infra,
// false if it is managed as a KKP Addon.
func IsManagedByAppInfra(cniType kubermaticv1.CNIPluginType, cniVersion string) bool {
	if cniType == kubermaticv1.CNIPluginTypeCilium {
		// Cilium is managed by the Applications infra starting from the version 1.13.0
		verConstraint, err := semverlib.NewConstraint(">= 1.13.0")
		if err != nil {
			return false
		}
		ver, err := semverlib.NewVersion(cniVersion)
		if err != nil {
			return false
		}
		if verConstraint.Check(ver) {
			return true
		}
	}
	return false
}
