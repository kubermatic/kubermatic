package main

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getAzureScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &azureScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &azureScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				ContainerLinux: &apimodels.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &azureScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type azureScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *azureScenario) Name() string {
	return fmt.Sprintf("azure-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *azureScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "azure-westeurope",
					Azure: &apimodels.AzureCloudSpec{
						ClientID:       secrets.Azure.ClientID,
						ClientSecret:   secrets.Azure.ClientSecret,
						SubscriptionID: secrets.Azure.SubscriptionID,
						TenantID:       secrets.Azure.TenantID,
					},
				},
				Version: s.version.String(),
			},
		},
	}
}

func (s *azureScenario) NodeDeployments(num int, _ secrets) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)
	size := "Standard_F2"
	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Azure: &apimodels.AzureNodeSpec{
							Size: &size,
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

func (s *azureScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
