package main

import (
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getDigitaloceanScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &digitaloceanScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv1.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &digitaloceanScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				ContainerLinux: &kubermaticapiv1.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &digitaloceanScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				CentOS: &kubermaticapiv1.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type digitaloceanScenario struct {
	version    *semver.Semver
	nodeOsSpec kubermaticapiv1.OperatingSystemSpec
}

func (s *digitaloceanScenario) Name() string {
	return fmt.Sprintf("digitalocean-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *digitaloceanScenario) Cluster(secrets secrets) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kubermaticv1.ClusterSpec{
			Version:           *s.version,
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
				DatacenterName: "do-ams3",
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					Token: secrets.Digitalocean.Token,
				},
			},
		},
	}
}

func (s *digitaloceanScenario) NodeDeployments(num int, _ kubermaticv1.DatacenterSpec, _ secrets) []kubermaticapiv1.NodeDeployment {
	return []kubermaticapiv1.NodeDeployment{
		{
			Spec: kubermaticapiv1.NodeDeploymentSpec{
				Replicas: int32(num),
				Template: kubermaticapiv1.NodeSpec{
					Cloud: kubermaticapiv1.NodeCloudSpec{
						Digitalocean: &kubermaticapiv1.DigitaloceanNodeSpec{
							Size: "4gb",
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

func (s *digitaloceanScenario) OS() kubermaticapiv1.OperatingSystemSpec {
	return s.nodeOsSpec
}
