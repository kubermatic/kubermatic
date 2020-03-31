/*
Copyright 2019 The Machine Controller Authors.

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

package types

import (
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
)

type RawConfig struct {
	AccessKeyID     providerconfigtypes.ConfigVarString `json:"accessKeyId,omitempty"`
	SecretAccessKey providerconfigtypes.ConfigVarString `json:"secretAccessKey,omitempty"`

	Region             providerconfigtypes.ConfigVarString   `json:"region"`
	AvailabilityZone   providerconfigtypes.ConfigVarString   `json:"availabilityZone,omitempty"`
	VpcID              providerconfigtypes.ConfigVarString   `json:"vpcId"`
	SubnetID           providerconfigtypes.ConfigVarString   `json:"subnetId"`
	SecurityGroupIDs   []providerconfigtypes.ConfigVarString `json:"securityGroupIDs,omitempty"`
	InstanceProfile    providerconfigtypes.ConfigVarString   `json:"instanceProfile,omitempty"`
	IsSpotInstance     *bool                                 `json:"isSpotInstance,omitempty"`
	InstanceType       providerconfigtypes.ConfigVarString   `json:"instanceType,omitempty"`
	AMI                providerconfigtypes.ConfigVarString   `json:"ami,omitempty"`
	DiskSize           int64                                 `json:"diskSize"`
	DiskType           providerconfigtypes.ConfigVarString   `json:"diskType,omitempty"`
	DiskIops           *int64                                `json:"diskIops,omitempty"`
	EBSVolumeEncrypted providerconfigtypes.ConfigVarBool     `json:"ebsVolumeEncrypted"`
	Tags               map[string]string                     `json:"tags,omitempty"`
	AssignPublicIP     *bool                                 `json:"assignPublicIP,omitempty"`
}
