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

package main

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

const alibabaDC = "alibaba-eu-central-1a"

// Returns a matrix of (version x operating system)
func getAlibabaScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &alibabaScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CentOS
		scenarios = append(scenarios, &alibabaScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type alibabaScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *alibabaScenario) Name() string {
	return fmt.Sprintf("alibaba-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *alibabaScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: alibabaDC,
					Alibaba: &apimodels.AlibabaCloudSpec{
						AccessKeySecret: secrets.Alibaba.AccessKeySecret,
						AccessKeyID:     secrets.Alibaba.AccessKeyID,
					},
				},
				Version: s.version,
			},
		},
	}
}

func (s *alibabaScenario) MachineDeployments(num int, secrets secrets) ([]apimodels.MachineDeployment, error) {
	return []apimodels.MachineDeployment{
		{
			Spec: &apimodels.MachineDeploymentSpec{
				Template: &apimodels.MachineSpec{
					Cloud: &apimodels.MachineCloudSpec{
						Alibaba: &apimodels.AlibabaMachineSpec{
							InstanceType:            "ecs.c6.xsmall",
							DiskSize:                "40",
							DiskType:                "cloud_efficiency",
							VSwitchID:               "vsw-gw8g8mn4ohmj483hsylmn",
							InternetMaxBandwidthOut: "10",
							ZoneID:                  alibabaDC,
						},
					},
					Versions: &apimodels.MachineVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.nodeOsSpec,
				},
			},
		},
	}, nil
}

func (s *alibabaScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
