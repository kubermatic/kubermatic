package main

import (
	"fmt"
	"strings"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getGCPScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &gcpScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv1.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &gcpScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				ContainerLinux: &kubermaticapiv1.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		//TODO: Fix
		//scenarios = append(scenarios, &gcpSenario{
		//	version: v,
		//	nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
		//		CentOS: &kubermaticapiv1.CentOSSpec{},
		//	},
		//})
	}
	return scenarios
}

type gcpScenario struct {
	version    *semver.Semver
	nodeOsSpec kubermaticapiv1.OperatingSystemSpec
}

func (s *gcpScenario) Name() string {
	version := strings.Replace(s.version.String(), ".", "-", -1)
	return fmt.Sprintf("gcp-%s-%s", getOSNameFromSpec(s.nodeOsSpec), version)
}

func (s *gcpScenario) Cluster(secrets secrets) *v1.Cluster {
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
				DatacenterName: "gcp-westeurope",
				GCP: &v1.GCPCloudSpec{
					ServiceAccount: secrets.GCP.ServiceAccount,
					Network:        secrets.GCP.Network,
					Subnetwork:     secrets.GCP.Subnetwork,
				},
			},
		},
	}
}

func (s *gcpScenario) Nodes(num int, secrets secrets) *kubermaticapiv1.NodeDeployment {
	return &kubermaticapiv1.NodeDeployment{
		Spec: kubermaticapiv1.NodeDeploymentSpec{
			Replicas: int32(num),
			Template: kubermaticapiv1.NodeSpec{
				Cloud: kubermaticapiv1.NodeCloudSpec{
					GCP: &kubermaticapiv1.GCPNodeSpec{
						Zone:        secrets.GCP.Zone,
						MachineType: "n1-standard-2",
						DiskType:    "pd-standard",
						DiskSize:    50,
						Preemptible: false,
						Labels: map[string]string{
							"kubernetes-cluster": "my-cluster",
						},
					},
				},
				Versions: kubermaticapiv1.NodeVersionInfo{
					Kubelet: s.version.String(),
				},
				OperatingSystem: s.nodeOsSpec,
			},
		},
	}
}
