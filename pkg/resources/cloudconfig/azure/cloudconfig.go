/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package azure

import (
	"encoding/json"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
)

type CloudConfig struct {
	Cloud           string `json:"cloud"`
	TenantID        string `json:"tenantId"`
	SubscriptionID  string `json:"subscriptionId"`
	AADClientID     string `json:"aadClientId"`
	AADClientSecret string `json:"aadClientSecret"`

	ResourceGroup              string `json:"resourceGroup"`
	Location                   string `json:"location"`
	VNetName                   string `json:"vnetName"`
	SubnetName                 string `json:"subnetName"`
	RouteTableName             string `json:"routeTableName"`
	SecurityGroupName          string `json:"securityGroupName" yaml:"securityGroupName"`
	PrimaryAvailabilitySetName string `json:"primaryAvailabilitySetName"`
	VnetResourceGroup          string `json:"vnetResourceGroup"`
	UseInstanceMetadata        bool   `json:"useInstanceMetadata"`
	LoadBalancerSku            string `json:"loadBalancerSku"`
}

func ForCluster(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, credentials resources.Credentials) CloudConfig {
	return CloudConfig{
		Cloud:                      "AZUREPUBLICCLOUD",
		TenantID:                   credentials.Azure.TenantID,
		SubscriptionID:             credentials.Azure.SubscriptionID,
		AADClientID:                credentials.Azure.ClientID,
		AADClientSecret:            credentials.Azure.ClientSecret,
		ResourceGroup:              cluster.Spec.Cloud.Azure.ResourceGroup,
		Location:                   dc.Spec.Azure.Location,
		VNetName:                   cluster.Spec.Cloud.Azure.VNetName,
		SubnetName:                 cluster.Spec.Cloud.Azure.SubnetName,
		RouteTableName:             cluster.Spec.Cloud.Azure.RouteTableName,
		SecurityGroupName:          cluster.Spec.Cloud.Azure.SecurityGroup,
		PrimaryAvailabilitySetName: cluster.Spec.Cloud.Azure.AvailabilitySet,
		VnetResourceGroup:          cluster.Spec.Cloud.Azure.VNetResourceGroup,
		UseInstanceMetadata:        false,
		LoadBalancerSku:            string(cluster.Spec.Cloud.Azure.LoadBalancerSKU),
	}
}

func (c *CloudConfig) String() (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	return string(b), nil
}
