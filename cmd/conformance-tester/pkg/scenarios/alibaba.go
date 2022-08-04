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
)

// GetAlibabaScenarios returns a matrix of (version x operating system).
func GetAlibabaScenarios(versions []*semver.Semver, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*alibabaScenario{
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

type alibabaScenario struct {
	baseScenario
}

func (s *alibabaScenario) DeepCopy() *alibabaScenario {
	return &alibabaScenario{
		baseScenario: *s.baseScenario.DeepCopy(),
	}
}

func (s *alibabaScenario) Name() string {
	return fmt.Sprintf("alibaba-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *alibabaScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.Alibaba.KKPDatacenter,
					Alibaba: &apimodels.AlibabaCloudSpec{
						AccessKeySecret: secrets.Alibaba.AccessKeySecret,
						AccessKeyID:     secrets.Alibaba.AccessKeyID,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *alibabaScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Alibaba.KKPDatacenter,
			Alibaba: &kubermaticv1.AlibabaCloudSpec{
				AccessKeySecret: secrets.Alibaba.AccessKeySecret,
				AccessKeyID:     secrets.Alibaba.AccessKeyID,
			},
		},
		Version: *s.version,
	}
}

func (s *alibabaScenario) NodeDeployments(_ context.Context, num int, secrets types.Secrets) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Alibaba: &apimodels.AlibabaNodeSpec{
							InstanceType:            "ecs.c6.xsmall",
							DiskSize:                "40",
							DiskType:                "cloud_efficiency",
							VSwitchID:               "vsw-gw8g8mn4ohmj483hsylmn",
							InternetMaxBandwidthOut: "10",
							ZoneID:                  s.getZoneID(),
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

func (s *alibabaScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	nodeSpec := apiv1.NodeSpec{
		Cloud: apiv1.NodeCloudSpec{
			Alibaba: &apiv1.AlibabaNodeSpec{
				InstanceType:            "ecs.c6.xsmall",
				DiskSize:                "40",
				DiskType:                "cloud_efficiency",
				VSwitchID:               "vsw-gw8g8mn4ohmj483hsylmn",
				InternetMaxBandwidthOut: "10",
				ZoneID:                  s.getZoneID(),
			},
		},
	}

	config, err := machine.GetAlibabaProviderConfig(cluster, nodeSpec, s.datacenter)
	if err != nil {
		return nil, err
	}

	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderAlibaba, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *alibabaScenario) getZoneID() string {
	return fmt.Sprintf("%sa", s.datacenter.Spec.Alibaba.Region)
}
