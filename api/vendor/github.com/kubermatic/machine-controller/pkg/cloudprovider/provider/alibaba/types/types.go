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
	AccessKeyID             providerconfigtypes.ConfigVarString `json:"accessKeyID,omitempty"`
	AccessKeySecret         providerconfigtypes.ConfigVarString `json:"accessKeySecret,omitempty"`
	RegionID                providerconfigtypes.ConfigVarString `json:"regionID,omitempty"`
	InstanceName            providerconfigtypes.ConfigVarString `json:"instanceName,omitempty"`
	InstanceType            providerconfigtypes.ConfigVarString `json:"instanceType,omitempty"`
	VSwitchID               providerconfigtypes.ConfigVarString `json:"vSwitchID,omitempty"`
	InternetMaxBandwidthOut providerconfigtypes.ConfigVarString `json:"internetMaxBandwidthOut,omitempty"`
	Labels                  map[string]string                   `json:"labels,omitempty"`
	ZoneID                  providerconfigtypes.ConfigVarString `json:"zoneID,omitempty"`
	DiskType                providerconfigtypes.ConfigVarString `json:"diskType,omitempty"`
	DiskSize                providerconfigtypes.ConfigVarString `json:"diskSize,omitempty"`
}
