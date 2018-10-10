package main

import (
	"fmt"

	"github.com/Masterminds/semver"

	kubermaticapiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getDigitaloceanScenarios() []testScenario {
	var scenarios []testScenario
	for _, v := range supportedVersions {
		// Ubuntu
		scenarios = append(scenarios, &digitaloceanScenario{
			version: v,
			nodeOsSpec: kubermaticapiv2.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv2.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &digitaloceanScenario{
			version: v,
			nodeOsSpec: kubermaticapiv2.OperatingSystemSpec{
				ContainerLinux: &kubermaticapiv2.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		//TODO: Fix
		//scenarios = append(scenarios, &digitaloceanScenario{
		//	version: v,
		//	nodeOsSpec: kubermaticapiv2.OperatingSystemSpec{
		//		CentOS: &kubermaticapiv2.CentOSSpec{},
		//	},
		//})
	}

	return scenarios
}

type digitaloceanScenario struct {
	version    *semver.Version
	nodeOsSpec kubermaticapiv2.OperatingSystemSpec
}

func (s *digitaloceanScenario) Name() string {
	return fmt.Sprintf("digitalocean-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *digitaloceanScenario) Cluster(secrets secrets) *v1.Cluster {
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
				DatacenterName: "do-fra1",
				Digitalocean: &v1.DigitaloceanCloudSpec{
					Token: secrets.Digitalocean.Token,
				},
			},
		},
	}
}

func (s *digitaloceanScenario) Nodes(num int) []*kubermaticapiv2.Node {
	var nodes []*kubermaticapiv2.Node
	for i := 0; i < num; i++ {
		node := &kubermaticapiv2.Node{
			Metadata: kubermaticapiv2.ObjectMeta{},
			Spec: kubermaticapiv2.NodeSpec{
				Cloud: kubermaticapiv2.NodeCloudSpec{
					Digitalocean: &kubermaticapiv2.DigitaloceanNodeSpec{
						Size: "4gb",
					},
				},
				Versions: kubermaticapiv2.NodeVersionInfo{
					Kubelet: s.version.String(),
				},
				OperatingSystem: s.nodeOsSpec,
			},
		}

		nodes = append(nodes, node)
	}

	return nodes
}
