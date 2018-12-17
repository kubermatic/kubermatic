package main

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/kubermatic/kubermatic/api/pkg/api/v2"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getOpenStackScenarios() []testScenario {
	var scenarios []testScenario
	for _, v := range supportedVersions {
		// Ubuntu
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			nodeOsSpec: v2.OperatingSystemSpec{
				Ubuntu: &v2.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			nodeOsSpec: v2.OperatingSystemSpec{
				ContainerLinux: &v2.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		//TODO: Fix
		//scenarios = append(scenarios, &openStackScenario{
		//	version: v,
		//	nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
		//		CentOS: &kubermaticapiv1.CentOSSpec{},
		//	},
		//})
	}

	return scenarios
}

type openStackScenario struct {
	version    *semver.Version
	nodeOsSpec v2.OperatingSystemSpec
}

func (s *openStackScenario) Name() string {
	return fmt.Sprintf("openstack-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *openStackScenario) Cluster(secrets secrets) *v1.Cluster {
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
				DatacenterName: "syseleven-dbl1",
				Openstack: &v1.OpenstackCloudSpec{
					Domain:   secrets.OpenStack.Domain,
					Tenant:   secrets.OpenStack.Tenant,
					Username: secrets.OpenStack.Username,
					Password: secrets.OpenStack.Password,
				},
			},
		},
	}
}

func (s *openStackScenario) Nodes(num int) []*v2.Node {
	osName := getOSNameFromSpec(s.nodeOsSpec)
	var nodes []*v2.Node
	for i := 0; i < num; i++ {
		node := &v2.Node{
			Metadata: v2.ObjectMeta{},
			Spec: v2.NodeSpec{
				Cloud: v2.NodeCloudSpec{
					Openstack: &v2.OpenstackNodeSpec{
						Flavor: "m1.small",
						Image:  "kubermatic-e2e-" + osName,
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
