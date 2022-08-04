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

package scenarios

import (
	"context"
	"fmt"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/machine"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	"k8s.io/utils/pointer"
)

const (
	nutanixCPUs     = 2
	nutanixMemoryMB = 4096
	nutanixDiskSize = 40
)

// GetNutanixScenarios returns a matrix of (version x operating system).
func GetNutanixScenarios(versions []*semver.Semver, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*nutanixScenario{
		{
			baseScenario: baseScenario{
				datacenter: datacenter,
				osSpec: apimodels.OperatingSystemSpec{
					Ubuntu: &apimodels.UbuntuSpec{},
				},
			},
		},
		{
			baseScenario: baseScenario{
				datacenter: datacenter,
				osSpec: apimodels.OperatingSystemSpec{
					Centos: &apimodels.CentOSSpec{},
				},
			},
		},
	}

	scenarios := []Scenario{}
	for _, v := range versions {
		for _, cri := range []string{resources.ContainerRuntimeContainerd, resources.ContainerRuntimeDocker} {
			for _, scenario := range baseScenarios {
				copy := scenario.DeepCopy()
				copy.version = v
				copy.containerRuntime = cri

				scenarios = append(scenarios, copy)
			}
		}
	}

	return scenarios
}

type nutanixScenario struct {
	baseScenario
}

func (s *nutanixScenario) DeepCopy() *nutanixScenario {
	return &nutanixScenario{
		baseScenario: *s.baseScenario.DeepCopy(),
	}
}

func (s *nutanixScenario) Name() string {
	return fmt.Sprintf("nutanix-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *nutanixScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	spec := &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.Nutanix.KKPDatacenter,
					Nutanix: &apimodels.NutanixCloudSpec{
						Username: secrets.Nutanix.Username,
						Password: secrets.Nutanix.Password,
						Csi: &apimodels.NutanixCSIConfig{
							Endpoint: secrets.Nutanix.CSIEndpoint,
							Password: secrets.Nutanix.CSIPassword,
							Username: secrets.Nutanix.CSIUsername,
						},
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

func (s *nutanixScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Nutanix.KKPDatacenter,
			Nutanix: &kubermaticv1.NutanixCloudSpec{
				Username: secrets.Nutanix.Username,
				Password: secrets.Nutanix.Password,
				CSI: &kubermaticv1.NutanixCSIConfig{
					Endpoint: secrets.Nutanix.CSIEndpoint,
					Password: secrets.Nutanix.CSIPassword,
					Username: secrets.Nutanix.CSIUsername,
				},
				ProxyURL:    secrets.Nutanix.ProxyURL,
				ClusterName: secrets.Nutanix.ClusterName,
				ProjectName: secrets.Nutanix.ProjectName,
			},
		},
		Version: *s.version,
	}
}

func (s *nutanixScenario) NodeDeployments(_ context.Context, num int, secrets types.Secrets) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.osSpec)
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Nutanix: &apimodels.NutanixNodeSpec{
							SubnetName: secrets.Nutanix.SubnetName,
							ImageName:  s.datacenter.Spec.Nutanix.Images[osName],
							CPUs:       nutanixCPUs,
							MemoryMB:   nutanixMemoryMB,
							DiskSize:   nutanixDiskSize,
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.osSpec,
				},
			},
		},
	}, nil
}

func (s *nutanixScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	os := getOSNameFromSpec(s.osSpec)

	nodeSpec := apiv1.NodeSpec{
		Cloud: apiv1.NodeCloudSpec{
			Nutanix: &apiv1.NutanixNodeSpec{
				SubnetName: secrets.Nutanix.SubnetName,
				ImageName:  s.datacenter.Spec.Nutanix.Images[os],
				CPUs:       nutanixCPUs,
				MemoryMB:   nutanixMemoryMB,
				DiskSize:   pointer.Int64(nutanixDiskSize),
			},
		},
	}

	config, err := machine.GetNutanixProviderConfig(cluster, nodeSpec, s.datacenter)
	if err != nil {
		return nil, err
	}

	md, err := createMachineDeployment(num, s.version, os, s.osSpec, providerconfig.CloudProviderNutanix, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
