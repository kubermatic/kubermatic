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

// RawConfig is a direct representation of an Azure machine object's configuration
type RawConfig struct {
	SubscriptionID providerconfigtypes.ConfigVarString `json:"subscriptionID,omitempty"`
	TenantID       providerconfigtypes.ConfigVarString `json:"tenantID,omitempty"`
	ClientID       providerconfigtypes.ConfigVarString `json:"clientID,omitempty"`
	ClientSecret   providerconfigtypes.ConfigVarString `json:"clientSecret,omitempty"`

	Location          providerconfigtypes.ConfigVarString `json:"location"`
	ResourceGroup     providerconfigtypes.ConfigVarString `json:"resourceGroup"`
	VMSize            providerconfigtypes.ConfigVarString `json:"vmSize"`
	VNetName          providerconfigtypes.ConfigVarString `json:"vnetName"`
	SubnetName        providerconfigtypes.ConfigVarString `json:"subnetName"`
	RouteTableName    providerconfigtypes.ConfigVarString `json:"routeTableName"`
	AvailabilitySet   providerconfigtypes.ConfigVarString `json:"availabilitySet"`
	SecurityGroupName providerconfigtypes.ConfigVarString `json:"securityGroupName"`

	ImageID        providerconfigtypes.ConfigVarString `json:"imageID"`
	OSDiskSize     int32                               `json:"osDiskSize"`
	DataDiskSize   int32                               `json:"dataDiskSize"`
	AssignPublicIP providerconfigtypes.ConfigVarBool   `json:"assignPublicIP"`
	Tags           map[string]string                   `json:"tags,omitempty"`
}
