package main

import (
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getPacketScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &packetScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv1.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &packetScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				ContainerLinux: &kubermaticapiv1.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &packetScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				CentOS: &kubermaticapiv1.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type packetScenario struct {
	version    *semver.Semver
	nodeOsSpec kubermaticapiv1.OperatingSystemSpec
}

func (s *packetScenario) Name() string {
	return fmt.Sprintf("packet-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *packetScenario) Cluster(secrets secrets) *v1.Cluster {
	return &v1.Cluster{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.ClusterSpec{
			Version:           *s.version,
			HumanReadableName: s.Name(),
			ClusterNetwork: v1.ClusterNetworkingConfig{
				Services: v1.NetworkRanges{
					CIDRBlocks: []string{"10.10.10.0/24"},
				},
				Pods: v1.NetworkRanges{
					CIDRBlocks: []string{"172.25.0.0/16"},
				},
				DNSDomain: "cluster.local",
			},
			Cloud: v1.CloudSpec{
				DatacenterName: "packet-ams1",
				Packet: &v1.PacketCloudSpec{
					APIKey:    secrets.Packet.APIKey,
					ProjectID: secrets.Packet.ProjectID,
				},
			},
		},
	}
}

func (s *packetScenario) Nodes(num int, _ provider.DatacenterSpec, _ secrets) []kubermaticapiv1.NodeDeployment {
	return []kubermaticapiv1.NodeDeployment{
		{
			Spec: kubermaticapiv1.NodeDeploymentSpec{
				Replicas: int32(num),
				Template: kubermaticapiv1.NodeSpec{
					Cloud: kubermaticapiv1.NodeCloudSpec{
						Packet: &kubermaticapiv1.PacketNodeSpec{
							InstanceType: "t1.small.x86",
						},
					},
					Versions: kubermaticapiv1.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: s.nodeOsSpec,
				},
			},
		},
	}
}

func (s *packetScenario) OS() kubermaticapiv1.OperatingSystemSpec {
	return s.nodeOsSpec
}
