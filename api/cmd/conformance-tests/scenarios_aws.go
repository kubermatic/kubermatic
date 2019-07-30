package main

import (
	"fmt"

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
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &awsScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				ContainerLinux: &apimodels.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		//TODO: This doesnt work for Kubernetes, fix
		scenarios = append(scenarios, &awsScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type awsScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *awsScenario) Name() string {
	return fmt.Sprintf("aws-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *awsScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
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

func (s *awsScenario) NodeDeployments(num int, secrets secrets) []apimodels.NodeDeployment {
	instanceType := "t2.medium"
	volumeType := "gp2"
	volumeSize := int64(100)

	nds := []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Aws: &apimodels.AWSNodeSpec{
							InstanceType:     &instanceType,
							VolumeType:       &volumeType,
							VolumeSize:       &volumeSize,
							AvailabilityZone: "eu-central-1a",
							SubnetID:         "subnet-2bff4f43",
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.nodeOsSpec,
				},
			},
		},
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Aws: &apimodels.AWSNodeSpec{
							InstanceType:     &instanceType,
							VolumeType:       &volumeType,
							VolumeSize:       &volumeSize,
							AvailabilityZone: "eu-central-1b",
							SubnetID:         "subnet-06d1167c",
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.nodeOsSpec,
				},
			},
		},
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Aws: &apimodels.AWSNodeSpec{
							InstanceType:     &instanceType,
							VolumeType:       &volumeType,
							VolumeSize:       &volumeSize,
							AvailabilityZone: "eu-central-1c",
							SubnetID:         "subnet-f3427db9",
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.nodeOsSpec,
				},
			},
		},
	}

	// evenly distribute the nodes among deployments
	nodesInEachAZ := num / 3
	azsWithExtraNode := num % 3

	for i := range nds {
		var replicas int32
		if i < azsWithExtraNode {
			replicas = int32(nodesInEachAZ + 1)
		} else {
			replicas = int32(nodesInEachAZ)
		}
		nds[i].Spec.Replicas = &replicas
	}

	return nds
}

func (s *awsScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
