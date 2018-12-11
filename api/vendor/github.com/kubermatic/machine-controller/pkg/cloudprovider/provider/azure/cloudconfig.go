package azure

import (
	"encoding/json"
	"fmt"
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
	UseInstanceMetadata        bool   `json:"useInstanceMetadata"`
}

func CloudConfigToString(c *CloudConfig) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return string(b), nil
}
