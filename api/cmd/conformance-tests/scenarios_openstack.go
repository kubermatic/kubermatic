package main

import (
	"fmt"
	"strconv"

	"github.com/Masterminds/semver"

	kubermaticapiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
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
			nodeOsSpec: kubermaticapiv2.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv2.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &openStackScenario{
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
		//scenarios = append(scenarios, &openStackScenario{
		//	version: v,
		//	nodeOsSpec: kubermaticapiv2.OperatingSystemSpec{
		//		CentOS: &kubermaticapiv2.CentOSSpec{},
		//	},
		//})
	}

	return scenarios
}

type openStackScenario struct {
	version    *semver.Version
	nodeOsSpec kubermaticapiv2.OperatingSystemSpec
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

func (s *openStackScenario) Nodes(num int) []*kubermaticapiv2.Node {
	osName := getOSNameFromSpec(s.nodeOsSpec)
	var nodes []*kubermaticapiv2.Node
	for i := 0; i < num; i++ {
		node := &kubermaticapiv2.Node{
			Metadata: kubermaticapiv2.ObjectMeta{
				Name: "node" + strconv.Itoa(i),
			},
			Spec: kubermaticapiv2.NodeSpec{
				Cloud: kubermaticapiv2.NodeCloudSpec{
					Openstack: &kubermaticapiv2.OpenstackNodeSpec{
						Flavor: "m1.small",
						Image:  "kubermatic-e2e-" + osName,
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
