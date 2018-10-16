package v1_test

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

func TestNewClusterSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	const valueToBeFiltered = "_______VALUE_TO_BE_FILTERED_______"

	cases := []struct {
		name    string
		cluster NewClusterSpec
	}{
		{
			"case 1: filter username and password from OpenStack",
			NewClusterSpec{
				Version: "1.2.3",
				Cloud: kubermaticv1.CloudSpec{
					DatacenterName: "OpenstackDatacenter",
					Openstack: &kubermaticv1.OpenstackCloudSpec{
						Username:       valueToBeFiltered,
						Password:       valueToBeFiltered,
						SubnetID:       "subnetID",
						Domain:         "domain",
						FloatingIPPool: "floatingIPPool",
						Network:        "network",
						RouterID:       "routerID",
						SecurityGroups: "securityGroups",
						Tenant:         "tenant",
					},
				},
			},
		},
		{
			"case 2: client ID and client secret from Azure",
			NewClusterSpec{
				Version: "1.2.3",
				Cloud: kubermaticv1.CloudSpec{
					Azure: &kubermaticv1.AzureCloudSpec{
						ClientID:        valueToBeFiltered,
						ClientSecret:    valueToBeFiltered,
						TenantID:        "tenantID",
						AvailabilitySet: "availablitySet",
						ResourceGroup:   "resourceGroup",
						RouteTableName:  "routeTableName",
						SecurityGroup:   "securityGroup",
						SubnetName:      "subnetName",
						SubscriptionID:  "subsciprionID",
						VNetName:        "vnetname",
					},
				},
			},
		},
		{
			"case 1: filter token from Hetzner",
			NewClusterSpec{
				Version: "1.2.3",
				Cloud: kubermaticv1.CloudSpec{
					Hetzner: &kubermaticv1.HetznerCloudSpec{
						Token: valueToBeFiltered,
					},
				},
			},
		},
		{
			"case 1: filter token from DigitalOcean",
			NewClusterSpec{
				Version: "1.2.3",
				Cloud: kubermaticv1.CloudSpec{
					Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
						Token: valueToBeFiltered,
					},
				},
			},
		},
		{
			"case 1: filter usernames and passwords from VSphere",
			NewClusterSpec{
				Version: "1.2.3",
				Cloud: kubermaticv1.CloudSpec{
					VSphere: &kubermaticv1.VSphereCloudSpec{
						Password: valueToBeFiltered,
						Username: valueToBeFiltered,
						InfraManagementUser: kubermaticv1.VSphereCredentials{
							Username: valueToBeFiltered,
							Password: valueToBeFiltered,
						},
						VMNetName: "vmNetName",
					},
				},
			},
		},
		{
			"case 1: filter access key ID and secret access key from AWS",
			NewClusterSpec{
				Version: "1.2.3",
				Cloud: kubermaticv1.CloudSpec{
					AWS: &kubermaticv1.AWSCloudSpec{
						AccessKeyID:         valueToBeFiltered,
						SecretAccessKey:     valueToBeFiltered,
						SecurityGroupID:     "secuirtyGroupID",
						AvailabilityZone:    "availablityZone",
						InstanceProfileName: "instanceProfileName",
						RoleName:            "roleName",
						RouteTableID:        "routeTableID",
						SubnetID:            "subnetID",
						VPCID:               "vpcID",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			jsonByteArray, err := c.cluster.MarshalJSON()
			if err != nil {
				t.Errorf("failed to marshal due to an error: %s", err)
			}

			if jsonString := string(jsonByteArray); strings.Contains(jsonString, valueToBeFiltered) {
				t.Errorf("output JSON: %s should not contain: %s", jsonString, valueToBeFiltered)
			}

			var jsonObject NewClusterSpec
			if err := json.Unmarshal(jsonByteArray, &jsonObject); err != nil {
				t.Errorf("failed to unmarshal due to an error: %s", err)
			}
		})
	}
}
