package main

import (
	"fmt"

	"github.com/Masterminds/semver"

	kubermaticapiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
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
			nodeOsSpec: kubermaticapiv2.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv2.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &vSphereScenario{
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
		//scenarios = append(scenarios, &vSphereScenario{
		//	version: v,
		//	nodeOsSpec: kubermaticapiv2.OperatingSystemSpec{
		//		CentOS: &kubermaticapiv2.CentOSSpec{},
		//	},
		//})
	}

	return scenarios
}

type vSphereScenario struct {
	version    *semver.Version
	nodeOsSpec kubermaticapiv2.OperatingSystemSpec
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

func (s *vSphereScenario) Nodes(num int) []*kubermaticapiv2.LegacyNode {
	osName := getOSNameFromSpec(s.nodeOsSpec)
	var nodes []*kubermaticapiv2.LegacyNode
	for i := 0; i < num; i++ {
		node := &kubermaticapiv2.LegacyNode{
			Metadata: kubermaticapiv2.LegacyObjectMeta{},
			Spec: kubermaticapiv2.NodeSpec{
				Cloud: kubermaticapiv2.NodeCloudSpec{
					VSphere: &kubermaticapiv2.VSphereNodeSpec{
						Template: fmt.Sprintf("%s-template", osName),
						CPUs:     2,
						Memory:   2048,
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
