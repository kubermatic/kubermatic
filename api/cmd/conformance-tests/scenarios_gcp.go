package main

import (
	"fmt"
	"strings"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
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
		scenarios = append(scenarios, &gcpScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				CentOS: &kubermaticapiv1.CentOSSpec{},
			},
		})
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

func (s *gcpScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "gcp-westeurope",
					Gcp: &apimodels.GCPCloudSpec{
						ServiceAccount: secrets.GCP.ServiceAccount,
						Network:        secrets.GCP.Network,
						Subnetwork:     secrets.GCP.Subnetwork,
					},
				},
				Version: s.version.String(),
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
