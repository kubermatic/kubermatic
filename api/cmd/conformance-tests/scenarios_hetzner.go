package main

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/kubermatic/kubermatic/api/pkg/api/v2"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getHetznerScenarios() []testScenario {
	var scenarios []testScenario
	for _, v := range supportedVersions {
		// Ubuntu
		scenarios = append(scenarios, &hetznerScenario{
			version: v,
			nodeOsSpec: v2.OperatingSystemSpec{
				Ubuntu: &v2.UbuntuSpec{},
			},
		})
		// CentOS
		//TODO: Fix
		//scenarios = append(scenarios, &hetznerScenario{
		//	version: v,
		//	nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
		//		CentOS: &kubermaticapiv1.CentOSSpec{},
		//	},
		//})
	}

	return scenarios
}

type hetznerScenario struct {
	version    *semver.Version
	nodeOsSpec v2.OperatingSystemSpec
}

func (s *hetznerScenario) Name() string {
	return fmt.Sprintf("hetzner-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *hetznerScenario) Cluster(secrets secrets) *v1.Cluster {
	return &v1.Cluster{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.ClusterSpec{
			Version:           s.version.String(),
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
				DatacenterName: "hetzner-fsn1",
				Hetzner: &v1.HetznerCloudSpec{
					Token: secrets.Hetzner.Token,
				},
			},
		},
	}
}

func (s *hetznerScenario) Nodes(num int) []*v2.Node {
	var nodes []*v2.Node
	for i := 0; i < num; i++ {
		node := &v2.Node{
			Metadata: v2.ObjectMeta{},
			Spec: v2.NodeSpec{
				Cloud: v2.NodeCloudSpec{
					Hetzner: &v2.HetznerNodeSpec{
						Type: "cx31",
					},
				},
				Versions: v2.NodeVersionInfo{
					Kubelet: s.version.String(),
				},
				OperatingSystem: s.nodeOsSpec,
			},
		}

		nodes = append(nodes, node)
	}

	return nodes
}
