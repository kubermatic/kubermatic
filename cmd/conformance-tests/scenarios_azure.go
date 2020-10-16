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

// Returns a matrix of (version x operating system)
func getAzureScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &azureScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &azureScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				ContainerLinux: &apimodels.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &azureScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type azureScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *azureScenario) Name() string {
	return fmt.Sprintf("azure-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *azureScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "azure-westeurope",
					Azure: &apimodels.AzureCloudSpec{
						ClientID:       secrets.Azure.ClientID,
						ClientSecret:   secrets.Azure.ClientSecret,
						SubscriptionID: secrets.Azure.SubscriptionID,
						TenantID:       secrets.Azure.TenantID,
					},
				},
				Version: s.version.String(),
			},
		},
	}
}

func (s *azureScenario) MachineDeployments(num int, _ secrets) ([]apimodels.MachineDeployment, error) {
	replicas := int32(num)
	size := "Standard_F2"
	return []apimodels.MachineDeployment{
		{
			Spec: &apimodels.MachineDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.MachineSpec{
					Cloud: &apimodels.MachineCloudSpec{
						Azure: &apimodels.AzureMachineSpec{
							Size: &size,
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

func (s *azureScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
