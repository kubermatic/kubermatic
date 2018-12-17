package main

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getVSphereScenarios() []testScenario {
	var scenarios []testScenario
	for _, v := range supportedVersions {
		// Ubuntu
		scenarios = append(scenarios, &vSphereScenario{
			version: v,
			nodeOsSpec: v2.OperatingSystemSpec{
				Ubuntu: &v2.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &vSphereScenario{
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
		//scenarios = append(scenarios, &vSphereScenario{
		//	version: v,
		//	nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
		//		CentOS: &kubermaticapiv1.CentOSSpec{},
		//	},
		//})
	}

	return scenarios
}

type vSphereScenario struct {
	version    *semver.Version
	nodeOsSpec v2.OperatingSystemSpec
}

func (s *vSphereScenario) Name() string {
	return fmt.Sprintf("vsphere-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *vSphereScenario) Cluster(secrets secrets) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kubermaticv1.ClusterSpec{
			Version:           s.version.String(),
			HumanReadableName: s.Name(),
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Services: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"10.10.10.0/24"},
				},
				Pods: kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{"172.25.0.0/16"},
				},
				DNSDomain: "cluster.local",
			},
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "vsphere-hetzner",
				VSphere: &kubermaticv1.VSphereCloudSpec{
					Username: secrets.VSphere.Username,
					Password: secrets.VSphere.Password,
					InfraManagementUser: kubermaticv1.VSphereCredentials{
						Username: secrets.VSphere.Username,
						Password: secrets.VSphere.Password,
					},
				},
			},
		},
	}
}

func (s *vSphereScenario) Nodes(num int) []*v2.Node {
	osName := getOSNameFromSpec(s.nodeOsSpec)
	var nodes []*v2.Node
	for i := 0; i < num; i++ {
		node := &v2.Node{
			Metadata: v2.ObjectMeta{},
			Spec: v2.NodeSpec{
				Cloud: v2.NodeCloudSpec{
					VSphere: &v2.VSphereNodeSpec{
						Template: fmt.Sprintf("%s-template", osName),
						CPUs:     2,
						Memory:   2048,
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
