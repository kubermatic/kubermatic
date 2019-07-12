package main

import (
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getOpenStackScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv1.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				ContainerLinux: &kubermaticapiv1.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				CentOS: &kubermaticapiv1.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type openStackScenario struct {
	version    *semver.Semver
	nodeOsSpec kubermaticapiv1.OperatingSystemSpec
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
						Domain:   secrets.OpenStack.Domain,
						Tenant:   secrets.OpenStack.Tenant,
						Username: secrets.OpenStack.Username,
						Password: secrets.OpenStack.Password,
					},
				},
			},
		},
	}
}

func (s *openStackScenario) NodeDeployments(num int, _ secrets) []kubermaticapiv1.NodeDeployment {
	osName := getOSNameFromSpec(s.nodeOsSpec)
	return []kubermaticapiv1.NodeDeployment{
		{
			Spec: kubermaticapiv1.NodeDeploymentSpec{
				Replicas: int32(num),
				Template: kubermaticapiv1.NodeSpec{
					Cloud: kubermaticapiv1.NodeCloudSpec{
						Openstack: &kubermaticapiv1.OpenstackNodeSpec{
							Flavor: "m1.small",
							Image:  "kubermatic-e2e-" + osName,
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

func (s *openStackScenario) OS() kubermaticapiv1.OperatingSystemSpec {
	return s.nodeOsSpec
}
