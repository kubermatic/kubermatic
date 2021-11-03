/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package aws

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

func newCloudProvider(t *testing.T) *AmazonEC2 {
	cs, err := getClientSet("test", "test", "eu-west-1", "http://localhost:4566")
	if err != nil {
		t.Fatalf("Failed to create AWS ClientSet: %v", err)
	}

	provider, err := NewCloudProvider(&kubermaticv1.Datacenter{
		Spec: kubermaticv1.DatacenterSpec{
			AWS: &kubermaticv1.DatacenterSpecAWS{
				Region: "eu-west-1",
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("failed to create cloud provider: %v", err)
	}

	provider.clientSet = cs

	return provider
}

func TestValidateCloudSpec(t *testing.T) {
	provider := newCloudProvider(t)

	defaultVPC, err := getDefaultVPC(provider.clientSet.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	defaultVPCID := *defaultVPC.VpcId

	// to properly test, we need the ID of a pre-existing security group
	sGroups, err := getSecurityGroupsWithClient(provider.clientSet.EC2)
	if err != nil {
		t.Fatalf("getSecurityGroupsWithClient should not have errored, but returned %v", err)
	}
	if len(sGroups) == 0 {
		t.Fatal("getSecurityGroupsWithClient should have found at least one security group")
	}

	securityGroupID := *sGroups[0].GroupId

	// NB: Remember that ValidateCloudSpec is not for validations
	// during regular reconciliations, but it is used to validate
	// the spec when a user creates a new cluster. That is why for
	// example a VPC ID must be given if a SG ID is given, because
	// at this point we never reconciled and have to rely solely on
	// the user input.

	testcases := []struct {
		name      string
		cloudSpec *kubermaticv1.AWSCloudSpec
		expectErr bool
	}{
		{
			name:      "empty-spec",
			cloudSpec: &kubermaticv1.AWSCloudSpec{},
			expectErr: false,
		},
		{
			name: "valid-vpc-id",
			cloudSpec: &kubermaticv1.AWSCloudSpec{
				VPCID: defaultVPCID,
			},
			expectErr: false,
		},
		{
			name: "valid-vpc-and-group",
			cloudSpec: &kubermaticv1.AWSCloudSpec{
				VPCID:           defaultVPCID,
				SecurityGroupID: securityGroupID,
			},
			expectErr: false,
		},
		{
			name: "invalid-vpc-id",
			cloudSpec: &kubermaticv1.AWSCloudSpec{
				VPCID: "does-not-exist",
			},
			expectErr: true,
		},
		{
			name: "no-vpc-given-but-required",
			cloudSpec: &kubermaticv1.AWSCloudSpec{
				SecurityGroupID: "does-not-exist",
			},
			expectErr: true,
		},
		{
			name: "valid-vpc-but-invalid-security-group",
			cloudSpec: &kubermaticv1.AWSCloudSpec{
				VPCID:           defaultVPCID,
				SecurityGroupID: "does-not-exist",
			},
			expectErr: true,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			err := provider.ValidateCloudSpec(kubermaticv1.CloudSpec{AWS: testcase.cloudSpec})
			if (err != nil) != testcase.expectErr {
				if testcase.expectErr {
					t.Error("Expected spec to fail, but no error was returned.")
				} else {
					t.Errorf("Expected spec to be valid, but error was returned: %v", err)
				}
			}
		})
	}
}

func TestInitializeCloudProvider(t *testing.T) {
	provider := newCloudProvider(t)
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})

	cluster, err := provider.InitializeCloudProvider(cluster, testClusterUpdater(cluster), false)
	if err != nil {
		t.Fatalf("InitializeCloudProvider should not have failed, but returned: %v", err)
	}

	if cluster.Spec.Cloud.AWS.VPCID == "" {
		t.Error("cloud spec should have a VPC ID")
	}

	if cluster.Spec.Cloud.AWS.RouteTableID == "" {
		t.Error("cloud spec should have a route table ID")
	}

	if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		t.Error("cloud spec should have a security group ID")
	}

	if cluster.Spec.Cloud.AWS.ControlPlaneRoleARN == "" {
		t.Error("cloud spec should have a control plane role name")
	}

	if cluster.Spec.Cloud.AWS.InstanceProfileName == "" {
		t.Error("cloud spec should have a instance profile name")
	}

	if cluster.Annotations[regionAnnotationKey] == "" {
		t.Error("cloud spec should have a region annotation")
	}
}
