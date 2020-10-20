/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"time"

	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils"
	awsapiclient "k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/client/aws"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"

	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

const awsDC = "aws-eu-central-1a"

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
		// Flatcar
		scenarios = append(scenarios, &awsScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Flatcar: &apimodels.FlatcarSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
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
					DatacenterName: awsDC,
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

func (s *awsScenario) NodeDeployments(num int, secrets secrets) ([]apimodels.NodeDeployment, error) {
	instanceType := "t2.medium"
	volumeType := "gp2"
	volumeSize := int64(100)

	listVPCParams := &awsapiclient.ListAWSVPCSParams{
		AccessKeyID:     utilpointer.StringPtr(secrets.AWS.AccessKeyID),
		SecretAccessKey: utilpointer.StringPtr(secrets.AWS.SecretAccessKey),
		DC:              awsDC,
	}
	utils.SetupParams(nil, listVPCParams, 5*time.Second, 1*time.Minute)

	vpcResponse, err := secrets.kubermaticClient.Aws.ListAWSVPCS(listVPCParams, secrets.kubermaticAuthenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to get vpcs: %v", err)
	}
	if len(vpcResponse.Payload) < 1 {
		return nil, errors.New("got zero vpcs back")
	}
	vpcID := vpcResponse.Payload[0].VpcID
	for _, vpc := range vpcResponse.Payload {
		if vpc.IsDefault {
			vpcID = vpc.VpcID
			break
		}
	}

	listSubnetParams := &awsapiclient.ListAWSSubnetsParams{
		AccessKeyID:     utilpointer.StringPtr(secrets.AWS.AccessKeyID),
		SecretAccessKey: utilpointer.StringPtr(secrets.AWS.SecretAccessKey),
		DC:              awsDC,
		VPC:             utilpointer.StringPtr(vpcID),
	}
	utils.SetupParams(nil, listSubnetParams, 5*time.Second, 1*time.Minute)

	subnetResponse, err := secrets.kubermaticClient.Aws.ListAWSSubnets(listSubnetParams, secrets.kubermaticAuthenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to get subnets: %v", err)
	}
	if n := len(subnetResponse.Payload); n < 3 {
		return nil, fmt.Errorf("expected to get at least three subnets, got %d", n)
	}

	// Find three subnets that are in different AZs to preserve the multi az testcase
	allAZs := sets.String{}
	var subnets []*apimodels.AWSSubnet
	for _, subnet := range subnetResponse.Payload {
		if !allAZs.Has(subnet.AvailabilityZone) {
			allAZs.Insert(subnet.AvailabilityZone)
			subnets = append(subnets, subnet)
		}
	}

	if n := len(subnets); n < 3 {
		return nil, fmt.Errorf("wanted three subnets in different AZs, got only %d", n)
	}

	nds := []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Aws: &apimodels.AWSNodeSpec{
							InstanceType:     &instanceType,
							VolumeType:       &volumeType,
							VolumeSize:       &volumeSize,
							AvailabilityZone: subnets[0].AvailabilityZone,
							SubnetID:         subnets[0].ID,
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
							AvailabilityZone: subnets[1].AvailabilityZone,
							SubnetID:         subnets[1].ID,
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
							AvailabilityZone: subnets[2].AvailabilityZone,
							SubnetID:         subnets[2].ID,
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

	return nds, nil
}

func (s *awsScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
