package main

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
	utilpointer "k8s.io/utils/pointer"
)

// Returns a matrix of (version x operating system)
func getKubevirtScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: &apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: &apimodels.OperatingSystemSpec{
				ContainerLinux: &apimodels.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: &apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type kubevirtScenario struct {
	version    *semver.Semver
	nodeOsSpec *apimodels.OperatingSystemSpec
}

func (s *kubevirtScenario) Name() string {
	return fmt.Sprintf("kubevirt-%s-%s", getOSNameFromSpec(*s.nodeOsSpec), s.version.String())
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

func (s *kubevirtScenario) NodeDeployments(num int, _ secrets) []apimodels.NodeDeployment {
	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: utilpointer.Int32Ptr(int32(num)),
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Kubevirt: &apimodels.KubevirtNodeSpec{
							Memory:           utilpointer.StringPtr("1024M"),
							Namespace:        utilpointer.StringPtr("kube-system"),
							SourceURL:        utilpointer.StringPtr("http://10.109.79.210/<< OS_NAME >>.img"),
							StorageClassName: utilpointer.StringPtr("kubermatic-fast"),
							PVCSize:          utilpointer.StringPtr("10Gi"),
							Cpus:             utilpointer.StringPtr("1"),
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: s.nodeOsSpec,
				},
			},
		},
	}
}

func (s *kubevirtScenario) OS() apimodels.OperatingSystemSpec {
	return *s.nodeOsSpec
}
