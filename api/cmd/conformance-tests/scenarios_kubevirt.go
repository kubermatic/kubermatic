package main

import (
	"fmt"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getKubevirtScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv1.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				ContainerLinux: &kubermaticapiv1.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				CentOS: &kubermaticapiv1.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type kubevirtScenario struct {
	version    *semver.Semver
	nodeOsSpec kubermaticapiv1.OperatingSystemSpec
}

func (s *kubevirtScenario) Name() string {
	return fmt.Sprintf("kubevirt-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *kubevirtScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					Kubevirt: &apimodels.KubevirtCloudSpec{
						Config: secrets.Kubevirt.Config,
					},
					DatacenterName: "kubevirt-ewr1",
				},
				Version: s.version.String(),
			},
		},
	}
}

func (s *kubevirtScenario) NodeDeployments(num int, _ secrets) []kubermaticapiv1.NodeDeployment {
	return []kubermaticapiv1.NodeDeployment{
		{
			Spec: kubermaticapiv1.NodeDeploymentSpec{
				Replicas: int32(num),
				Template: kubermaticapiv1.NodeSpec{
					Cloud: kubermaticapiv1.NodeCloudSpec{
						Kubevirt: &kubermaticapiv1.KubevirtNodeSpec{
							Memory:           "1024M",
							Namespace:        "kube-system",
							SourceURL:        "http://10.109.79.210/<< OS_NAME >>.img",
							StorageClassName: "kubermatic-fast",
							PVCSize:          "10Gi",
							CPUs:             "1",
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

func (s *kubevirtScenario) OS() kubermaticapiv1.OperatingSystemSpec {
	return s.nodeOsSpec
}
