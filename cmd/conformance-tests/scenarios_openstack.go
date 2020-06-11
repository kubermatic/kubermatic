package main

import (
	"fmt"

	"github.com/kubermatic/kubermatic/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getOpenStackScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				ContainerLinux: &apimodels.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type openStackScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *openStackScenario) Name() string {
	return fmt.Sprintf("openstack-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *openStackScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "syseleven-dbl1",
					Openstack: &apimodels.OpenstackCloudSpec{
						Domain:         secrets.OpenStack.Domain,
						Tenant:         secrets.OpenStack.Tenant,
						Username:       secrets.OpenStack.Username,
						Password:       secrets.OpenStack.Password,
						FloatingIPPool: "ext-net",
					},
				},
				Version: s.version.String(),
			},
		},
	}
}

func (s *openStackScenario) NodeDeployments(num int, _ secrets) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.nodeOsSpec)
	image := "kubermatic-e2e-" + osName
	flavor := "m1.small"
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Openstack: &apimodels.OpenstackNodeSpec{
							Flavor: &flavor,
							Image:  &image,
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

func (s *openStackScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
