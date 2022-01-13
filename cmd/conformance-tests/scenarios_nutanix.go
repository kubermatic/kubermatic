/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"context"
	"fmt"
	"strings"

	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getNutanixScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &nutanixScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CentOS
		scenarios = append(scenarios, &nutanixScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type nutanixScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *nutanixScenario) Name() string {
	return fmt.Sprintf("nutanix-%s-%s", getOSNameFromSpec(s.nodeOsSpec), strings.ReplaceAll(s.version.String(), ".", "-"))
}

func (s *nutanixScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	spec := &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "nutanix-ger",
					Nutanix: &apimodels.NutanixCloudSpec{
						Username:    secrets.Nutanix.Username,
						Password:    secrets.Nutanix.Password,
						ProxyURL:    secrets.Nutanix.ProxyURL,
						ClusterName: secrets.Nutanix.ClusterName,
						ProjectName: secrets.Nutanix.ProjectName,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}

	return spec
}

func (s *nutanixScenario) NodeDeployments(_ context.Context, num int, secrets secrets) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.nodeOsSpec)
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Nutanix: &apimodels.NutanixNodeSpec{
							SubnetName: secrets.Nutanix.SubnetName,
							ImageName:  fmt.Sprintf("machine-controller-e2e-%s", osName),
							CPUs:       2,
							MemoryMB:   4096,
							DiskSize:   40,
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.nodeOsSpec,
				},
			},
		},
	}, nil
}

func (s *nutanixScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
