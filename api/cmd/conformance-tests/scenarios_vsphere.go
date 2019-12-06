package main

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getVSphereScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &vSphereScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &vSphereScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				ContainerLinux: &apimodels.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &vSphereScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type vSphereScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *vSphereScenario) Name() string {
	return fmt.Sprintf("vsphere-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *vSphereScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "vsphere-ger",
					Vsphere: &apimodels.VSphereCloudSpec{
						Username: secrets.VSphere.Username,
						Password: secrets.VSphere.Password,
					},
				},
				Version: s.version.String(),
			},
		},
	}
}

func (s *vSphereScenario) NodeDeployments(num int, _ secrets) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.nodeOsSpec)
	replicas := int32(num)
	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Vsphere: &apimodels.VSphereNodeSpec{
							Template: fmt.Sprintf("machine-controller-e2e-%s", osName),
							Cpus:     2,
							Memory:   4096,
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

func (s *vSphereScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
