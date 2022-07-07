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
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

const (
	packetDatacenter = "packet-ewr1"
)

// GetPacketScenarios returns a matrix of (version x operating system).
func GetPacketScenarios(versions []*semver.Semver) []Scenario {
	var scenarios []Scenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &packetScenario{
			version:      v,
			metro:        "TY",
			instanceType: "m3.small.x86",
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CentOS
		scenarios = append(scenarios, &packetScenario{
			version:      v,
			metro:        "AM",
			instanceType: "c3.small.x86",
			osSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type packetScenario struct {
	version      *semver.Semver
	metro        string
	instanceType string
	osSpec       apimodels.OperatingSystemSpec
}

func (s *packetScenario) Name() string {
	return fmt.Sprintf("packet-%s-%s", getOSNameFromSpec(s.osSpec), s.version.String())
}

func (s *packetScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: packetDatacenter,
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
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: packetDatacenter,
			Packet: &kubermaticv1.PacketCloudSpec{
				APIKey:    secrets.Packet.APIKey,
				ProjectID: secrets.Packet.ProjectID,
			},
		},
		Version: *s.version,
	}
}

func (s *packetScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)
	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Packet: &apimodels.PacketNodeSpec{
							InstanceType: &s.instanceType,
							Metro:        s.metro,
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
		InstanceType: providerconfig.ConfigVarString{Value: s.instanceType},
	})
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *packetScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}
