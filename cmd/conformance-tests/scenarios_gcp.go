package main

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getGCPScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &gcpScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &gcpScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				ContainerLinux: &apimodels.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &gcpScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type gcpScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
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

func (s *gcpScenario) NodeDeployments(num int, secrets secrets) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Gcp: &apimodels.GCPNodeSpec{
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
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.nodeOsSpec,
				},
			},
		},
	}, nil
}

func (s *gcpScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
