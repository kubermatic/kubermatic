/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package scenarios

import (
	"context"
	"errors"
	"fmt"
	"time"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/go-openapi/runtime"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/resources/machine"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	apiclient "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client"
	awsapiclient "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/aws"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
	utilpointer "k8s.io/utils/pointer"
)

const (
	awsInstanceType = "t2.medium"
	awsVolumeType   = "gp2"
	awsVolumeSize   = 100
)

type awsScenario struct {
	baseScenario

	kubermaticClient        *apiclient.KubermaticKubernetesPlatformAPI
	kubermaticAuthenticator runtime.ClientAuthInfoWriter
}

func (s *awsScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.AWS.KKPDatacenter,
					Aws: &apimodels.AWSCloudSpec{
						SecretAccessKey: secrets.AWS.SecretAccessKey,
						AccessKeyID:     secrets.AWS.AccessKeyID,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *awsScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.AWS.KKPDatacenter,
			AWS: &kubermaticv1.AWSCloudSpec{
				SecretAccessKey: secrets.AWS.SecretAccessKey,
				AccessKeyID:     secrets.AWS.AccessKeyID,
			},
		},
		Version: s.version,
	}
}

func (s *awsScenario) NodeDeployments(ctx context.Context, replicas int, secrets types.Secrets) ([]apimodels.NodeDeployment, error) {
	instanceType := awsInstanceType
	volumeType := awsVolumeType
	volumeSize := int32(awsVolumeSize)

	listVPCParams := &awsapiclient.ListAWSVPCSParams{
		Context:         ctx,
		AccessKeyID:     utilpointer.StringPtr(secrets.AWS.AccessKeyID),
		SecretAccessKey: utilpointer.StringPtr(secrets.AWS.SecretAccessKey),
		DC:              secrets.AWS.KKPDatacenter,
	}
	utils.SetupParams(nil, listVPCParams, 5*time.Second, 1*time.Minute)

	vpcResponse, err := s.kubermaticClient.Aws.ListAWSVPCS(listVPCParams, s.kubermaticAuthenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to get vpcs: %w", err)
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
		Context:         ctx,
		AccessKeyID:     utilpointer.StringPtr(secrets.AWS.AccessKeyID),
		SecretAccessKey: utilpointer.StringPtr(secrets.AWS.SecretAccessKey),
		DC:              secrets.AWS.KKPDatacenter,
		VPC:             utilpointer.StringPtr(vpcID),
	}
	utils.SetupParams(nil, listSubnetParams, 5*time.Second, 1*time.Minute)

	subnetResponse, err := s.kubermaticClient.Aws.ListAWSSubnets(listSubnetParams, s.kubermaticAuthenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to get subnets: %w", err)
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

	osSpec, err := s.APIOperatingSystemSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to build OS spec: %w", err)
	}

	nds := []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Aws: &apimodels.AWSNodeSpec{
							InstanceType:         &instanceType,
							VolumeType:           &volumeType,
							VolumeSize:           &volumeSize,
							AvailabilityZone:     subnets[0].AvailabilityZone,
							SubnetID:             subnets[0].ID,
							IsSpotInstance:       true,
							SpotInstanceMaxPrice: "0.5", // USD
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: osSpec,
				},
			},
		},
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Aws: &apimodels.AWSNodeSpec{
							InstanceType:         &instanceType,
							VolumeType:           &volumeType,
							VolumeSize:           &volumeSize,
							AvailabilityZone:     subnets[1].AvailabilityZone,
							SubnetID:             subnets[1].ID,
							IsSpotInstance:       true,
							SpotInstanceMaxPrice: "0.5", // USD
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: osSpec,
				},
			},
		},
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Aws: &apimodels.AWSNodeSpec{
							InstanceType:         &instanceType,
							VolumeType:           &volumeType,
							VolumeSize:           &volumeSize,
							AvailabilityZone:     subnets[2].AvailabilityZone,
							SubnetID:             subnets[2].ID,
							IsSpotInstance:       true,
							SpotInstanceMaxPrice: "0.5", // USD
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: osSpec,
				},
			},
		},
	}

	// evenly distribute the nodes among deployments
	nodesInEachAZ := replicas / 3
	azsWithExtraNode := replicas % 3

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

func (s *awsScenario) MachineDeployments(ctx context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	vpcs, err := awsprovider.GetVPCS(ctx, secrets.AWS.AccessKeyID, secrets.AWS.SecretAccessKey, "", "", s.datacenter.Spec.AWS.Region)
	if err != nil {
		return nil, err
	}

	if len(vpcs) == 0 {
		return nil, errors.New("no VPCs found")
	}

	vpcID := vpcs[0].VpcId
	for _, vpc := range vpcs {
		if vpc.IsDefault != nil && *vpc.IsDefault {
			vpcID = vpc.VpcId
			break
		}
	}

	allSubnets, err := awsprovider.GetSubnets(ctx, secrets.AWS.AccessKeyID, secrets.AWS.SecretAccessKey, "", "", s.datacenter.Spec.AWS.Region, *vpcID)
	if err != nil {
		return nil, err
	}
	if n := len(allSubnets); n < 3 {
		return nil, fmt.Errorf("expected to get at least three subnets, got %d", n)
	}

	// Find three subnets that are in different AZs to preserve the multi az testcase
	allAZs := sets.NewString()
	var subnets []ec2types.Subnet
	for _, subnet := range allSubnets {
		if !allAZs.Has(pointer.StringDeref(subnet.AvailabilityZone, "")) {
			allAZs.Insert(pointer.StringDeref(subnet.AvailabilityZone, ""))
			subnets = append(subnets, subnet)
		}
	}

	if n := len(subnets); n < 3 {
		return nil, fmt.Errorf("wanted three subnets in different AZs, got only %d", n)
	}

	result := []clusterv1alpha1.MachineDeployment{}

	osSpec, err := s.OperatingSystemSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to build OS spec: %w", err)
	}

	for _, subnet := range subnets {
		nodeSpec := apiv1.NodeSpec{
			OperatingSystem: *osSpec,
			Cloud: apiv1.NodeCloudSpec{
				AWS: &apiv1.AWSNodeSpec{
					InstanceType:         awsInstanceType,
					VolumeType:           awsVolumeType,
					VolumeSize:           awsVolumeSize,
					AvailabilityZone:     *subnet.AvailabilityZone,
					SubnetID:             *subnet.SubnetId,
					IsSpotInstance:       pointer.Bool(true),
					SpotInstanceMaxPrice: pointer.String("0.5"), // USD
				},
			},
		}

		config, err := machine.GetAWSProviderConfig(cluster, nodeSpec, s.datacenter)
		if err != nil {
			return nil, err
		}

		md, err := s.createMachineDeployment(num, config)
		if err != nil {
			return nil, err
		}

		result = append(result, md)
	}

	// evenly distribute the nodes among deployments
	nodesInEachAZ := num / 3
	azsWithExtraNode := num % 3

	for i := range result {
		var replicas int32
		if i < azsWithExtraNode {
			replicas = int32(nodesInEachAZ + 1)
		} else {
			replicas = int32(nodesInEachAZ)
		}
		result[i].Spec.Replicas = &replicas
	}

	return result, nil
}
