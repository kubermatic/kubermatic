package main

import (
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getAWSScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &awsScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv1.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &awsScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				ContainerLinux: &kubermaticapiv1.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		//TODO: This doesnt work for Kubernetes, fix
		scenarios = append(scenarios, &awsScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				CentOS: &kubermaticapiv1.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type awsScenario struct {
	version    *semver.Semver
	nodeOsSpec kubermaticapiv1.OperatingSystemSpec
}

func (s *awsScenario) Name() string {
	return fmt.Sprintf("aws-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *awsScenario) Cluster(secrets secrets) *kubermaticv1.Cluster {
	return nil
}

func (s *awsScenario) APICluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "aws-eu-central-1a",
					Aws: &apimodels.AWSCloudSpec{
						SecretAccessKey: secrets.AWS.SecretAccessKey,
						AccessKeyID:     secrets.AWS.AccessKeyID,
					},
				},
				Version: s.version,
			},
		},
	}
}

func (s *awsScenario) NodeDeployments(num int, _ secrets) []kubermaticapiv1.NodeDeployment {
	nds := []kubermaticapiv1.NodeDeployment{
		{
			Spec: kubermaticapiv1.NodeDeploymentSpec{
				Template: kubermaticapiv1.NodeSpec{
					Cloud: kubermaticapiv1.NodeCloudSpec{
						AWS: &kubermaticapiv1.AWSNodeSpec{
							InstanceType:     "t2.medium",
							VolumeType:       "gp2",
							VolumeSize:       100,
							AvailabilityZone: "eu-central-1a",
							SubnetID:         "subnet-2bff4f43",
						},
					},
					Versions: kubermaticapiv1.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: s.nodeOsSpec,
				},
			},
		},
		{
			Spec: kubermaticapiv1.NodeDeploymentSpec{
				Template: kubermaticapiv1.NodeSpec{
					Cloud: kubermaticapiv1.NodeCloudSpec{
						AWS: &kubermaticapiv1.AWSNodeSpec{
							InstanceType:     "t2.medium",
							VolumeType:       "gp2",
							VolumeSize:       100,
							AvailabilityZone: "eu-central-1b",
							SubnetID:         "subnet-06d1167c",
						},
					},
					Versions: kubermaticapiv1.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: s.nodeOsSpec,
				},
			},
		},
		{
			Spec: kubermaticapiv1.NodeDeploymentSpec{
				Template: kubermaticapiv1.NodeSpec{
					Cloud: kubermaticapiv1.NodeCloudSpec{
						AWS: &kubermaticapiv1.AWSNodeSpec{
							InstanceType:     "t2.medium",
							VolumeType:       "gp2",
							VolumeSize:       100,
							AvailabilityZone: "eu-central-1c",
							SubnetID:         "subnet-f3427db9",
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

	// evenly distribute the nodes among deployments
	nodesInEachAZ := num / 3
	azsWithExtraNode := num % 3

	for i := range nds {
		if i < azsWithExtraNode {
			nds[i].Spec.Replicas = int32(nodesInEachAZ + 1)
		} else {
			nds[i].Spec.Replicas = int32(nodesInEachAZ)
		}
	}

	return nds
}

func (s *awsScenario) OS() kubermaticapiv1.OperatingSystemSpec {
	return s.nodeOsSpec
}
