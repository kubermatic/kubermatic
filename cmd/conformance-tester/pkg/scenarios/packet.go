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
	"errors"
	"fmt"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	equinixmetaltypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/equinixmetal/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

const (
	packetInstanceType = "m3.small.x86"
)

// GetPacketScenarios returns a matrix of (version x operating system).
func GetPacketScenarios(versions []*semver.Semver, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*packetScenario{
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

type packetScenario struct {
	baseScenario
}

func (s *packetScenario) DeepCopy() *packetScenario {
	return &packetScenario{
		baseScenario: *s.baseScenario.DeepCopy(),
	}
}

func (s *packetScenario) Name() string {
	return fmt.Sprintf("packet-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *packetScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.Packet.KKPDatacenter,
					Packet: &apimodels.PacketCloudSpec{
						APIKey:    secrets.Packet.APIKey,
						ProjectID: secrets.Packet.ProjectID,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *packetScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Packet.KKPDatacenter,
			Packet: &kubermaticv1.PacketCloudSpec{
				APIKey:    secrets.Packet.APIKey,
				ProjectID: secrets.Packet.ProjectID,
			},
		},
		Version: *s.version,
	}
}

func (s *packetScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets) ([]apimodels.NodeDeployment, error) {
	instanceType := packetInstanceType
	replicas := int32(num)
	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Packet: &apimodels.PacketNodeSpec{
							InstanceType: &instanceType,
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

func (s *packetScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	// See alibaba provider for more info on this.
	return nil, errors.New("not implemented for gitops yet")

	//nolint:govet
	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderPacket, equinixmetaltypes.RawConfig{
		InstanceType: providerconfig.ConfigVarString{Value: packetInstanceType},
	})
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
