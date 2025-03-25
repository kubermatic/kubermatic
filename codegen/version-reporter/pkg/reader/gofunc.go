/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package reader

import (
	"fmt"

	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/resources/dns"
	"k8c.io/kubermatic/v2/pkg/resources/konnectivity"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/resources/kubernetes-dashboard"
	"k8c.io/kubermatic/v2/pkg/version"
)

type versionProviderFunc func(clusterVersion semver.Semver) (string, error)

var versionProviderFuncs = map[string]versionProviderFunc{
	"callAWSCCMVersion": func(clusterVersion semver.Semver) (string, error) {
		return cloudcontroller.AWSCCMVersion(clusterVersion), nil
	},
	"callAzureCCMVersion": cloudcontroller.AzureCCMVersion,
	"callDigitaloceanCCMVersion": func(clusterVersion semver.Semver) (string, error) {
		return cloudcontroller.DigitaloceanCCMVersion(clusterVersion), nil
	},
	"callGCPCCMVersion": func(clusterVersion semver.Semver) (string, error) {
		return cloudcontroller.GCPCCMVersion(clusterVersion), nil
	},
	"callOpenStackCCMVersion": cloudcontroller.OpenStackCCMTag,
	"callVSphereCCMVersion": func(clusterVersion semver.Semver) (string, error) {
		return cloudcontroller.VSphereCCMVersion(clusterVersion), nil
	},
	"callCoreDNSVersion": func(clusterVersion semver.Semver) (string, error) {
		return dns.CoreDNSVersion(clusterVersion.Semver()), nil
	},
	"callKonnectivityVersion": func(clusterVersion semver.Semver) (string, error) {
		return konnectivity.NetworkProxyVersion(clusterVersion), nil
	},
	"callKubernetesDashboardVersion": kubernetesdashboard.DashboardVersion,
}

func CallGoFunction(function string) (map[string]string, error) {
	fun, ok := versionProviderFuncs[function]
	if !ok {
		return nil, fmt.Errorf("unknown function %q", function)
	}

	clusterVersions := []semver.Semver{}

	for _, version := range version.GetLatestMinorVersions(defaulting.DefaultKubernetesVersioning.Versions) {
		clusterVersions = append(clusterVersions, *semver.NewSemverOrDie(version))
	}

	result := map[string]string{}
	for _, clusterVersion := range clusterVersions {
		componentVersion, err := fun(clusterVersion)
		if err != nil {
			return nil, fmt.Errorf("failed calling %q for %v: %w", function, clusterVersion, err)
		}

		result[clusterVersion.MajorMinor()] = componentVersion
	}

	return result, nil
}
