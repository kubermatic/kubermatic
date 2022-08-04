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

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/go-openapi/runtime"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	awstypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	apiclient "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client"
	awsapiclient "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/aws"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

const (
	awsInstanceType = "t2.medium"
	awsVolumeType   = "gp2"
	awsVolumeSize   = 100
)

// GetAWSScenarios returns a matrix of (version x operating system).
func GetAWSScenarios(versions []*semver.Semver, kubermaticClient *apiclient.KubermaticKubernetesPlatformAPI, kubermaticAuthenticator runtime.ClientAuthInfoWriter, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*awsScenario{
		{
			datacenter:              datacenter.Spec.AWS,
			kubermaticClient:        kubermaticClient,
			kubermaticAuthenticator: kubermaticAuthenticator,
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		},
		{
			datacenter:              datacenter.Spec.AWS,
			kubermaticClient:        kubermaticClient,
			kubermaticAuthenticator: kubermaticAuthenticator,
			osSpec: apimodels.OperatingSystemSpec{
				Flatcar: &apimodels.FlatcarSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		},
		{
			datacenter:              datacenter.Spec.AWS,
			kubermaticClient:        kubermaticClient,
			kubermaticAuthenticator: kubermaticAuthenticator,
			osSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		},
		{
			datacenter:              datacenter.Spec.AWS,
			kubermaticClient:        kubermaticClient,
			kubermaticAuthenticator: kubermaticAuthenticator,
			osSpec: apimodels.OperatingSystemSpec{
				Rhel: &apimodels.RHELSpec{},
			},
		},
		{
			datacenter:              datacenter.Spec.AWS,
			kubermaticClient:        kubermaticClient,
			kubermaticAuthenticator: kubermaticAuthenticator,
			osSpec: apimodels.OperatingSystemSpec{
				RockyLinux: &apimodels.RockyLinuxSpec{},
			},
		},
	}

	scenarios := []Scenario{}
	for _, v := range versions {
		for _, cri := range []string{resources.ContainerRuntimeContainerd, resources.ContainerRuntimeDocker} {
			for _, scenario := range baseScenarios {
				copy := scenario.DeepCopy()
				copy.version = v
				copy.containerRuntime = cri

				scenarios = append(scenarios, copy)
			}
		}
	}

	return scenarios
}

type awsScenario struct {
	version                 *semver.Semver
	containerRuntime        string
	datacenter              *kubermaticv1.DatacenterSpecAWS
	osSpec                  apimodels.OperatingSystemSpec
	kubermaticClient        *apiclient.KubermaticKubernetesPlatformAPI
	kubermaticAuthenticator runtime.ClientAuthInfoWriter
}

func (s *awsScenario) DeepCopy() *awsScenario {
	version := s.version.DeepCopy()

	return &awsScenario{
		version:                 &version,
		containerRuntime:        s.containerRuntime,
		datacenter:              s.datacenter.DeepCopy(),
		osSpec:                  s.osSpec,
		kubermaticClient:        s.kubermaticClient,
		kubermaticAuthenticator: s.kubermaticAuthenticator,
	}
}

func (s *awsScenario) ContainerRuntime() string {
	return s.containerRuntime
}

func (s *awsScenario) Name() string {
	return fmt.Sprintf("aws-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
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
		Version: *s.version,
	}
}

func (s *awsScenario) NodeDeployments(
	ctx context.Context,
	num int,
	secrets types.Secrets,
) ([]apimodels.NodeDeployment, error) {
	instanceType := awsInstanceType
	volumeType := awsVolumeType
	volumeSize := int64(awsVolumeSize)

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
					OperatingSystem: &s.osSpec,
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
					OperatingSystem: &s.osSpec,
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
					OperatingSystem: &s.osSpec,
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

func (s *awsScenario) MachineDeployments(ctx context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	vpcs, err := awsprovider.GetVPCS(ctx, secrets.AWS.AccessKeyID, secrets.AWS.SecretAccessKey, "", "", s.datacenter.Region)
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

	allSubnets, err := awsprovider.GetSubnets(ctx, secrets.AWS.AccessKeyID, secrets.AWS.SecretAccessKey, "", "", s.datacenter.Region, *vpcID)
	if err != nil {
		return nil, err
	}
	if n := len(allSubnets); n < 3 {
		return nil, fmt.Errorf("expected to get at least three subnets, got %d", n)
	}

	// Find three subnets that are in different AZs to preserve the multi az testcase
	allAZs := sets.NewString()
	var subnets []*ec2.Subnet
	for _, subnet := range allSubnets {
		if !allAZs.Has(*subnet.AvailabilityZone) {
			allAZs.Insert(*subnet.AvailabilityZone)
			subnets = append(subnets, subnet)
		}
	}

	if n := len(subnets); n < 3 {
		return nil, fmt.Errorf("wanted three subnets in different AZs, got only %d", n)
	}

	result := []clusterv1alpha1.MachineDeployment{}

	for _, subnet := range subnets {
		ami := s.datacenter.Images[getOSNameFromSpec(s.osSpec)]

		config := awstypes.RawConfig{
			AMI:              providerconfig.ConfigVarString{Value: ami},
			InstanceType:     providerconfig.ConfigVarString{Value: awsInstanceType},
			DiskType:         providerconfig.ConfigVarString{Value: awsVolumeType},
			DiskSize:         int64(awsVolumeSize),
			AvailabilityZone: providerconfig.ConfigVarString{Value: *subnet.AvailabilityZone},
			Region:           providerconfig.ConfigVarString{Value: s.datacenter.Region},
			VpcID:            providerconfig.ConfigVarString{Value: *vpcID},
			SubnetID:         providerconfig.ConfigVarString{Value: *subnet.SubnetId},
			// rely on the KKP's reconciling to have filled these fields in already and
			// the caller to have since refreshed the cluster object
			InstanceProfile: providerconfig.ConfigVarString{Value: cluster.Spec.Cloud.AWS.InstanceProfileName},
			SecurityGroupIDs: []providerconfig.ConfigVarString{
				{Value: cluster.Spec.Cloud.AWS.SecurityGroupID},
			},
		}

		config.Tags = map[string]string{}
		config.Tags["kubernetes.io/cluster/"+cluster.Name] = ""
		config.Tags["system/cluster"] = cluster.Name

		projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
		if ok {
			config.Tags["system/project"] = projectID
		}

		md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderAWS, config)
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

func (s *awsScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}
