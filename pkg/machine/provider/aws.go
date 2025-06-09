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

package provider

import (
	"fmt"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/aws"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/utils/ptr"
)

const (
	awsDefaultDiskType           = string(ec2types.VolumeTypeGp2)
	awsDefaultDiskSize           = 25
	awsDefaultEBSVolumeEncrypted = false
)

type awsConfig struct {
	aws.RawConfig
}

func NewAWSConfig() *awsConfig {
	return &awsConfig{}
}

func (b *awsConfig) Build() aws.RawConfig {
	return b.RawConfig
}

func (b *awsConfig) WithRegion(region string) *awsConfig {
	b.Region.Value = region
	return b
}

func (b *awsConfig) WithAvailabilityZone(az string) *awsConfig {
	b.AvailabilityZone.Value = az
	return b
}

func (b *awsConfig) WithVpcID(vpcID string) *awsConfig {
	b.VpcID.Value = vpcID
	return b
}

func (b *awsConfig) WithSubnetID(subnet string) *awsConfig {
	b.SubnetID.Value = subnet
	return b
}

func (b *awsConfig) WithSecurityGroupID(sgID string) *awsConfig {
	b.SecurityGroupIDs = []providerconfig.ConfigVarString{{
		Value: sgID,
	}}
	return b
}

func (b *awsConfig) WithAMI(ami string) *awsConfig {
	b.AMI.Value = ami
	return b
}

func (b *awsConfig) WithInstanceProfile(profileName string) *awsConfig {
	b.InstanceProfile.Value = profileName
	return b
}

func (b *awsConfig) WithInstanceType(itype string) *awsConfig {
	b.InstanceType.Value = itype
	return b
}

func (b *awsConfig) WithDiskType(diskType string) *awsConfig {
	b.DiskType.Value = diskType
	return b
}

func (b *awsConfig) WithDiskSize(diskSize int) *awsConfig {
	b.DiskSize = int32(diskSize)
	return b
}

func (b *awsConfig) WithDiskIops(iops int) *awsConfig {
	b.DiskIops = ptr.To[int32](int32(iops))
	return b
}

func (b *awsConfig) WithAssignPublicIP(assign bool) *awsConfig {
	b.AssignPublicIP = ptr.To(assign)
	return b
}

func (b *awsConfig) WithEBSVolumeEncrypted(encrypted bool) *awsConfig {
	b.EBSVolumeEncrypted.Value = ptr.To(encrypted)
	return b
}

func (b *awsConfig) WithTag(tagKey string, tagValue string) *awsConfig {
	if b.Tags == nil {
		b.Tags = map[string]string{}
	}
	b.Tags[tagKey] = tagValue
	return b
}

func (b *awsConfig) WithSpotInstanceMaxPrice(maxPrice string) *awsConfig {
	if maxPrice == "" {
		b.IsSpotInstance = ptr.To(false)
		b.SpotInstanceConfig = nil
	} else {
		b.IsSpotInstance = ptr.To(true)
		b.SpotInstanceConfig = &aws.SpotInstanceConfig{
			MaxPrice: providerconfig.ConfigVarString{Value: maxPrice},
		}
	}

	return b
}

func CompleteAWSProviderSpec(config *aws.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecAWS, os providerconfig.OperatingSystem) (*aws.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.AWS == nil {
		return nil, fmt.Errorf("cannot use cluster to create AWS cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if config == nil {
		config = &aws.RawConfig{}
	}

	if config.DiskType.Value == "" {
		config.DiskType.Value = awsDefaultDiskType
	}

	if config.DiskSize == 0 {
		config.DiskSize = awsDefaultDiskSize
	}

	if config.EBSVolumeEncrypted.Value == nil {
		config.EBSVolumeEncrypted.Value = ptr.To(awsDefaultEBSVolumeEncrypted)
	}

	if datacenter != nil {
		if config.Region.Value == "" {
			config.Region.Value = datacenter.Region
		}

		if config.AMI.Value == "" && os != "" {
			// This can still be empty, but that's okay, the machine-controller will later, during
			// reconciliations, default the AMI for us.
			config.AMI.Value = datacenter.Images[os]
		}
	}

	if config.AvailabilityZone.Value == "" && config.Region.Value != "" {
		config.AvailabilityZone.Value = fmt.Sprintf("%sa", config.Region.Value)
	}

	if cluster != nil {
		if config.VpcID.Value == "" {
			config.VpcID.Value = cluster.Spec.Cloud.AWS.VPCID
		}

		if config.InstanceProfile.Value == "" {
			config.InstanceProfile.Value = cluster.Spec.Cloud.AWS.InstanceProfileName
		}

		securityGroupIDExists := false
		// Check if a valid SecurityGroupID exists.
		for _, sec := range config.SecurityGroupIDs {
			if !IsConfigVarStringEmpty(sec) {
				securityGroupIDExists = true
				break
			}
		}

		if !securityGroupIDExists {
			config.SecurityGroupIDs = []providerconfig.ConfigVarString{{
				Value: cluster.Spec.Cloud.AWS.SecurityGroupID,
			}}
		}

		if config.Tags == nil {
			config.Tags = map[string]string{}
		}

		config.Tags["system/cluster"] = cluster.Name
		config.Tags["kubernetes.io/cluster/"+cluster.Name] = ""

		if projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; ok {
			config.Tags["system/project"] = projectID
		}

		config.AMI = providerconfig.ConfigVarString{
			Value: "ami-028727bd3039c5a1f",
		}
	}

	return config, nil
}
