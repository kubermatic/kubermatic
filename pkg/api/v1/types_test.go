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

package v1_test

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/semver"
)

func TestNewClusterSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	const valueToBeFiltered = "_______VALUE_TO_BE_FILTERED_______"

	cases := []struct {
		name    string
		cluster ClusterSpec
	}{
		{
			"case 1: filter username and password from OpenStack",
			ClusterSpec{
				Version: *semver.NewSemverOrDie("1.2.3"),
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
			ClusterSpec{
				Version: *semver.NewSemverOrDie("1.2.3"),
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
			"case 3: filter token from Hetzner",
			ClusterSpec{
				Version: *semver.NewSemverOrDie("1.2.3"),
				Cloud: kubermaticv1.CloudSpec{
					Hetzner: &kubermaticv1.HetznerCloudSpec{
						Token: valueToBeFiltered,
					},
				},
			},
		},
		{
			"case 4: filter token from DigitalOcean",
			ClusterSpec{
				Version: *semver.NewSemverOrDie("1.2.3"),
				Cloud: kubermaticv1.CloudSpec{
					Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
						Token: valueToBeFiltered,
					},
				},
			},
		},
		{
			"case 5: filter usernames and passwords from VSphere",
			ClusterSpec{
				Version: *semver.NewSemverOrDie("1.2.3"),
				Cloud: kubermaticv1.CloudSpec{
					VSphere: &kubermaticv1.VSphereCloudSpec{
						Password: valueToBeFiltered,
						Username: valueToBeFiltered,
						InfraManagementUser: kubermaticv1.VSphereCredentials{
							Username: valueToBeFiltered,
							Password: valueToBeFiltered,
						},
						VMNetName: "vmNetName",
						Datastore: "testDataStore",
					},
				},
			},
		},
		{
			"case 6: filter access key ID and secret access key from AWS",
			ClusterSpec{
				Version: *semver.NewSemverOrDie("1.2.3"),
				Cloud: kubermaticv1.CloudSpec{
					AWS: &kubermaticv1.AWSCloudSpec{
						AccessKeyID:         valueToBeFiltered,
						SecretAccessKey:     valueToBeFiltered,
						SecurityGroupID:     "secuirtyGroupID",
						InstanceProfileName: "instanceProfileName",
						RoleName:            "roleName",
						RouteTableID:        "routeTableID",
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

			var jsonObject ClusterSpec
			if err := json.Unmarshal(jsonByteArray, &jsonObject); err != nil {
				t.Errorf("failed to unmarshal due to an error: %s", err)
			}
		})
	}
}
