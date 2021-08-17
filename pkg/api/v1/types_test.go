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

	. "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
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

func TestDigitalOceanNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *DigitaloceanNodeSpec
		expected string
	}{
		{
			"case 1: should fail when size is not provided",
			&DigitaloceanNodeSpec{},
			"missing or invalid required parameter(s): size",
		},
		{
			"case 2: should marshal when size is provided",
			&DigitaloceanNodeSpec{
				Size: "test-size",
			},
			"{\"size\":\"test-size\",\"backups\":false,\"ipv6\":false,\"monitoring\":false,\"tags\":null}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestHetznerNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *HetznerNodeSpec
		expected string
	}{
		{
			"case 1: should fail when type is not provided",
			&HetznerNodeSpec{},
			"missing or invalid required parameter(s): type",
		},
		{
			"case 2: should marshal when type is provided",
			&HetznerNodeSpec{
				Type:    "test-type",
				Network: "test",
			},
			"{\"network\":\"test\",\"type\":\"test-type\"}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestAzureNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *AzureNodeSpec
		expected string
	}{
		{
			"case 1: should fail when size is not provided",
			&AzureNodeSpec{},
			"missing or invalid required parameter(s): size",
		},
		{
			"case 2: should marshal when size is provided",
			&AzureNodeSpec{
				Size: "test-size",
			},
			"{\"size\":\"test-size\",\"assignPublicIP\":false,\"osDiskSize\":0,\"dataDiskSize\":0,\"zones\":null,\"imageID\":\"\",\"assignAvailabilitySet\":false}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestVSphereNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *VSphereNodeSpec
		expected string
	}{
		{
			"case 1: should fail when required parameters are not provided",
			&VSphereNodeSpec{},
			"missing or invalid required parameter(s): cpus, memory, diskSizeGB, template",
		},
		{
			"case 2: should fail when only cpus are provided",
			&VSphereNodeSpec{
				CPUs: 1,
			},
			"missing or invalid required parameter(s): memory, diskSizeGB, template",
		},
		{
			"case 3: should fail when cpus and memory are provided",
			&VSphereNodeSpec{
				CPUs:   1,
				Memory: 1,
			},
			"missing or invalid required parameter(s): diskSizeGB, template",
		},
		{
			"case 4: should fail when cpus, memory and disk size are provided",
			&VSphereNodeSpec{
				CPUs:       1,
				Memory:     1,
				DiskSizeGB: &[]int64{1}[0],
			},
			"missing or invalid required parameter(s): template",
		},
		{
			"case 5: should fail when cpus count is wrong",
			&VSphereNodeSpec{
				CPUs:       0,
				Memory:     1,
				DiskSizeGB: &[]int64{1}[0],
				Template:   "test-template",
			},
			"missing or invalid required parameter(s): cpus",
		},
		{
			"case 6: should fail when memory count is wrong",
			&VSphereNodeSpec{
				CPUs:       1,
				Memory:     0,
				DiskSizeGB: &[]int64{1}[0],
				Template:   "test-template",
			},
			"missing or invalid required parameter(s): memory",
		},
		{
			"case 7: should fail when disk size is wrong",
			&VSphereNodeSpec{
				CPUs:       1,
				Memory:     1,
				DiskSizeGB: &[]int64{0}[0],
				Template:   "test-template",
			},
			"missing or invalid required parameter(s): diskSizeGB",
		},
		{
			"case 8: should marshal when all required parameters are provided",
			&VSphereNodeSpec{
				CPUs:       1,
				Memory:     1,
				DiskSizeGB: &[]int64{1}[0],
				Template:   "test-template",
			},
			"{\"cpus\":1,\"memory\":1,\"diskSizeGB\":1,\"template\":\"test-template\"}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestOpenstackNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *OpenstackNodeSpec
		expected string
	}{
		{
			"case 1: should fail when required parameters are not provided",
			&OpenstackNodeSpec{},
			"missing or invalid required parameter(s): flavor, image",
		},
		{
			"case 2: should fail when only flavor is provided",
			&OpenstackNodeSpec{
				Flavor: "test-flavor",
			},
			"missing or invalid required parameter(s): image",
		},
		{
			"case 3: should fail when only image is provided",
			&OpenstackNodeSpec{
				Image: "test-image",
			},
			"missing or invalid required parameter(s): flavor",
		},
		{
			"case 4: should marshal when all required parameters are provided",
			&OpenstackNodeSpec{
				Flavor: "test-flavor",
				Image:  "test-image",
			},
			"{\"flavor\":\"test-flavor\",\"image\":\"test-image\",\"diskSize\":null,\"availabilityZone\":\"\",\"instanceReadyCheckPeriod\":\"\",\"instanceReadyCheckTimeout\":\"\"}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestAWSNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *AWSNodeSpec
		expected string
	}{
		{
			"case 1: should fail when required parameters are not provided",
			&AWSNodeSpec{},
			"missing or invalid required parameter(s): instanceType, diskSize, volumeType",
		},
		{
			"case 2: should fail when only instance type is provided",
			&AWSNodeSpec{
				InstanceType: "test-instance",
			},
			"missing or invalid required parameter(s): diskSize, volumeType",
		},
		{
			"case 3: should fail when only volume type is provided",
			&AWSNodeSpec{
				VolumeType: "test-volume",
			},
			"missing or invalid required parameter(s): instanceType, diskSize",
		},
		{
			"case 4: should fail when only volume size is provided",
			&AWSNodeSpec{
				VolumeSize: 1,
			},
			"missing or invalid required parameter(s): instanceType, volumeType",
		},
		{
			"case 3: should fail when volume size is wrong",
			&AWSNodeSpec{
				VolumeSize:   0,
				InstanceType: "test-instance",
				VolumeType:   "test-volume",
			},
			"missing or invalid required parameter(s): diskSize",
		},
		{
			"case 4: should marshal when all required parameters are provided",
			&AWSNodeSpec{
				InstanceType: "test-instance",
				VolumeSize:   1,
				VolumeType:   "test-volume",
			},
			"{\"instanceType\":\"test-instance\",\"diskSize\":1,\"volumeType\":\"test-volume\",\"ami\":\"\",\"tags\":null,\"availabilityZone\":\"\",\"subnetID\":\"\",\"assignPublicIP\":null,\"isSpotInstance\":null}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestPacketNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *PacketNodeSpec
		expected string
	}{
		{
			"case 1: should fail when instance type is not provided",
			&PacketNodeSpec{},
			"missing or invalid required parameter(s): instanceType",
		},
		{
			"case 2: should marshal when instance type is provided",
			&PacketNodeSpec{
				InstanceType: "test-instance",
			},
			"{\"instanceType\":\"test-instance\",\"tags\":null}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestGCPNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *GCPNodeSpec
		expected string
	}{
		{
			"case 1: should fail when required parameters are not provided",
			&GCPNodeSpec{},
			"missing or invalid required parameter(s): zone, diskSize, machineType, diskType",
		},
		{
			"case 2: should fail when only zone is provided",
			&GCPNodeSpec{
				Zone: "test-zone",
			},
			"missing or invalid required parameter(s): diskSize, machineType, diskType",
		},
		{
			"case 3: should fail when only diskSize is provided",
			&GCPNodeSpec{
				DiskSize: 1,
			},
			"missing or invalid required parameter(s): zone, machineType, diskType",
		},
		{
			"case 4: should fail when only machineType is provided",
			&GCPNodeSpec{
				MachineType: "test-machine",
			},
			"missing or invalid required parameter(s): zone, diskSize, diskType",
		},
		{
			"case 5: should fail when only diskType is provided",
			&GCPNodeSpec{
				DiskType: "test-disk",
			},
			"missing or invalid required parameter(s): zone, diskSize, machineType",
		},
		{
			"case 6: should fail when diskSize is invalid",
			&GCPNodeSpec{
				DiskSize: 0,
			},
			"missing or invalid required parameter(s): zone, diskSize, machineType, diskType",
		},
		{
			"case 7: should marshal when instance type is provided",
			&GCPNodeSpec{
				Zone:        "test-zone",
				MachineType: "test-machine",
				DiskSize:    1,
				DiskType:    "test-disk",
			},
			"{\"zone\":\"test-zone\",\"machineType\":\"test-machine\",\"diskSize\":1,\"diskType\":\"test-disk\",\"preemptible\":false,\"labels\":null,\"tags\":null,\"customImage\":\"\"}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestKubevirtNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *KubevirtNodeSpec
		expected string
	}{
		{
			"case 1: should fail when required parameters are not provided",
			&KubevirtNodeSpec{},
			"missing or invalid required parameter(s): cpus, memory, namespace, sourceURL, storageClassName, pvcSize",
		},
		{
			"case 2: should fail when only cpus is provided",
			&KubevirtNodeSpec{
				CPUs: "1",
			},
			"missing or invalid required parameter(s): memory, namespace, sourceURL, storageClassName, pvcSize",
		},
		{
			"case 3: should fail when only memory is provided",
			&KubevirtNodeSpec{
				Memory: "1",
			},
			"missing or invalid required parameter(s): cpus, namespace, sourceURL, storageClassName, pvcSize",
		},
		{
			"case 4: should fail when only namespace is provided",
			&KubevirtNodeSpec{
				Namespace: "test-ns",
			},
			"missing or invalid required parameter(s): cpus, memory, sourceURL, storageClassName, pvcSize",
		},
		{
			"case 5: should fail when only sourceURL is provided",
			&KubevirtNodeSpec{
				SourceURL: "test-url",
			},
			"missing or invalid required parameter(s): cpus, memory, namespace, storageClassName, pvcSize",
		},
		{
			"case 6: should fail when only storageClassName is provided",
			&KubevirtNodeSpec{
				StorageClassName: "test-sc",
			},
			"missing or invalid required parameter(s): cpus, memory, namespace, sourceURL, pvcSize",
		},
		{
			"case 7: should fail when only pvcSize is provided",
			&KubevirtNodeSpec{
				PVCSize: "1",
			},
			"missing or invalid required parameter(s): cpus, memory, namespace, sourceURL, storageClassName",
		},
		{
			"case 8: should marshal when instance type is provided",
			&KubevirtNodeSpec{
				CPUs:             "1",
				Memory:           "1",
				Namespace:        "test-ns",
				SourceURL:        "test-url",
				StorageClassName: "test-sc",
				PVCSize:          "1",
			},
			"{\"cpus\":\"1\",\"memory\":\"1\",\"namespace\":\"test-ns\",\"sourceURL\":\"test-url\",\"storageClassName\":\"test-sc\",\"pvcSize\":\"1\"}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestAlibabaNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *AlibabaNodeSpec
		expected string
	}{
		{
			"case 1: should fail when required parameters are not provided",
			&AlibabaNodeSpec{},
			"missing or invalid required parameter(s): instanceType, diskSize, diskType, vSwitchID, internetMaxBandwidthOut, zoneID",
		},
		{
			"case 2: should fail when only instanceType is provided",
			&AlibabaNodeSpec{
				InstanceType: "test-instance",
			},
			"missing or invalid required parameter(s): diskSize, diskType, vSwitchID, internetMaxBandwidthOut, zoneID",
		},
		{
			"case 3: should fail when only diskSize is provided",
			&AlibabaNodeSpec{
				DiskSize: "1",
			},
			"missing or invalid required parameter(s): instanceType, diskType, vSwitchID, internetMaxBandwidthOut, zoneID",
		},
		{
			"case 4: should fail when only diskType is provided",
			&AlibabaNodeSpec{
				DiskType: "test-disk",
			},
			"missing or invalid required parameter(s): instanceType, diskSize, vSwitchID, internetMaxBandwidthOut, zoneID",
		},
		{
			"case 5: should fail when only vSwitchID is provided",
			&AlibabaNodeSpec{
				VSwitchID: "test-vswitch",
			},
			"missing or invalid required parameter(s): instanceType, diskSize, diskType, internetMaxBandwidthOut, zoneID",
		},
		{
			"case 6: should fail when only internetMaxBandwidthOut is provided",
			&AlibabaNodeSpec{
				InternetMaxBandwidthOut: "1",
			},
			"missing or invalid required parameter(s): instanceType, diskSize, diskType, vSwitchID, zoneID",
		},
		{
			"case 7: should fail when only zoneID is provided",
			&AlibabaNodeSpec{
				ZoneID: "test-zone",
			},
			"missing or invalid required parameter(s): instanceType, diskSize, diskType, vSwitchID, internetMaxBandwidthOut",
		},
		{
			"case 8: should marshal when instance type is provided",
			&AlibabaNodeSpec{
				InstanceType:            "test-instance",
				DiskSize:                "1",
				DiskType:                "test-disk",
				VSwitchID:               "test-vswitch",
				InternetMaxBandwidthOut: "1",
				ZoneID:                  "test-zone",
			},
			"{\"instanceType\":\"test-instance\",\"diskSize\":\"1\",\"diskType\":\"test-disk\",\"vSwitchID\":\"test-vswitch\",\"internetMaxBandwidthOut\":\"1\",\"labels\":null,\"zoneID\":\"test-zone\"}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}

func TestAnexiaNodeSpec_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		spec     *AnexiaNodeSpec
		expected string
	}{
		{
			"case 1: should fail when required parameters are not provided",
			&AnexiaNodeSpec{},
			"missing or invalid required parameter(s): vlanID, cpus, memory, diskSize, templateID",
		},
		{
			"case 2: should fail when only vlanID is provided",
			&AnexiaNodeSpec{
				VlanID: "test-vlan",
			},
			"missing or invalid required parameter(s): cpus, memory, diskSize, templateID",
		},
		{
			"case 3: should fail when only templateID is provided",
			&AnexiaNodeSpec{
				TemplateID: "test-template",
			},
			"missing or invalid required parameter(s): vlanID, cpus, memory, diskSize",
		},
		{
			"case 4: should fail when only cpus is provided",
			&AnexiaNodeSpec{
				CPUs: 1,
			},
			"missing or invalid required parameter(s): vlanID, memory, diskSize, templateID",
		},
		{
			"case 5: should fail when only memory is provided",
			&AnexiaNodeSpec{
				Memory: 1,
			},
			"missing or invalid required parameter(s): vlanID, cpus, diskSize, templateID",
		},
		{
			"case 6: should fail when only diskSize is provided",
			&AnexiaNodeSpec{
				DiskSize: 1,
			},
			"missing or invalid required parameter(s): vlanID, cpus, memory, templateID",
		},
		{
			"case 8: should marshal when instance type is provided",
			&AnexiaNodeSpec{
				VlanID:     "test-vlan",
				TemplateID: "test-template",
				CPUs:       1,
				Memory:     1,
				DiskSize:   1,
			},
			"{\"vlanID\":\"test-vlan\",\"templateID\":\"test-template\",\"cpus\":1,\"memory\":1,\"diskSize\":1}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			marshalledBytes, err := json.Marshal(c.spec)
			if err != nil && !strings.Contains(err.Error(), c.expected) {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, err.Error())
			}

			if len(marshalledBytes) > 0 && string(marshalledBytes) != c.expected {
				t.Errorf("expected: %v,\nbut got: %v", c.expected, string(marshalledBytes))
			}
		})
	}
}
