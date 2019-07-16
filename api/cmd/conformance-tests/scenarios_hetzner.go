package main

import (
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getHetznerScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &hetznerScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv1.UbuntuSpec{},
			},
		})
		// CentOS
		scenarios = append(scenarios, &hetznerScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				CentOS: &kubermaticapiv1.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type hetznerScenario struct {
	version    *semver.Semver
	nodeOsSpec kubermaticapiv1.OperatingSystemSpec
}

func (s *hetznerScenario) Name() string {
	return fmt.Sprintf("hetzner-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *hetznerScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "hetzner-nbg1",
					Hetzner: &apimodels.HetznerCloudSpec{
						Token: secrets.Hetzner.Token,
					},
				},
				Version: s.version.String(),
			},
		},
	}
}

func (s *hetznerScenario) Nodes(num int, _ secrets) *kubermaticapiv1.NodeDeployment {
	return &kubermaticapiv1.NodeDeployment{
		Spec: kubermaticapiv1.NodeDeploymentSpec{
			Replicas: int32(num),
			Template: kubermaticapiv1.NodeSpec{
				Cloud: kubermaticapiv1.NodeCloudSpec{
					Hetzner: &kubermaticapiv1.HetznerNodeSpec{
						Type: "cx31",
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
