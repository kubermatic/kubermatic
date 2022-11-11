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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

// CanalCNILastUnspecifiedVersion is the last Canal CNI version applied in KKP user-clusters
// started in KKP before v2.18 release. If cluster.spec.cniPlugin is not set, it means Canal of this version.
const CanalCNILastUnspecifiedVersion = "v3.8"

var (
	defaultCNIPluginVersion = map[kubermaticv1.CNIPluginType]string{
		kubermaticv1.CNIPluginTypeCanal:  "v3.23",
		kubermaticv1.CNIPluginTypeCilium: "v1.12",
	}
)

var (
	// supportedCNIPlugins contains a lis of all currently supported CNI Plugin types.
	supportedCNIPlugins = sets.NewString(
		kubermaticv1.CNIPluginTypeCanal.String(),
		kubermaticv1.CNIPluginTypeCilium.String(),
		kubermaticv1.CNIPluginTypeNone.String(),
	)
	// supportedCNIPluginVersions contains a list of all currently supported CNI versions for each CNI type.
	// Only supported versions are available for selection in KKP UI.
	supportedCNIPluginVersions = map[kubermaticv1.CNIPluginType]sets.String{
		kubermaticv1.CNIPluginTypeCanal:  sets.NewString("v3.20", "v3.21", "v3.22", "v3.23"),
		kubermaticv1.CNIPluginTypeCilium: sets.NewString("v1.11", "v1.12", "v1.13"),
		kubermaticv1.CNIPluginTypeNone:   sets.NewString(""),
	}
	// deprecatedCNIPluginVersions contains a list of deprecated CNI versions for each CNI type.
	// Deprecated versions are not available for selection in KKP UI, but are still accepted
	// by the validation webhook for backward compatibility.
	deprecatedCNIPluginVersions = map[kubermaticv1.CNIPluginType]sets.String{
		kubermaticv1.CNIPluginTypeCanal: sets.NewString("v3.8", "v3.19"),
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
func GetSupportedCNIPlugins() sets.String {
	return supportedCNIPlugins
}

// GetSupportedCNIPluginVersions returns currently supported CNI versions for a CNI type.
func GetSupportedCNIPluginVersions(cniPluginType kubermaticv1.CNIPluginType) (sets.String, error) {
	if !supportedCNIPlugins.Has(cniPluginType.String()) {
		return sets.NewString(), fmt.Errorf("CNI Plugin type %q not supported. Supported types %s", cniPluginType, supportedCNIPlugins.List())
	}

	versions, ok := supportedCNIPluginVersions[cniPluginType]
	// this means we messed up, should not happen as we support the plugin above
	if !ok {
		return sets.NewString(), fmt.Errorf("no versions available for CNI plugin %q", cniPluginType)
	}

	return versions, nil
}

// GetAllowedCNIPluginVersions returns all allowed CNI versions for a CNI type (supported + deprecated).
func GetAllowedCNIPluginVersions(cniPluginType kubermaticv1.CNIPluginType) (sets.String, error) {
	supported, err := GetSupportedCNIPluginVersions(cniPluginType)
	if err != nil {
		return sets.NewString(), err
	}
	allowed := sets.NewString(supported.List()...)
	if deprecated, ok := deprecatedCNIPluginVersions[cniPluginType]; ok {
		allowed.Insert(deprecated.List()...)
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
		if cniVersion == "v1.12" { // TODO: use smever to check >= 1.13
			return true
		}
		return false
	}
	return false
}
