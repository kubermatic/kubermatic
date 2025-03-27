//go:build integration

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
	"context"
	"fmt"
	"os"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
)

func newCloudProvider(t *testing.T) *AmazonEC2 {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	provider, err := NewCloudProvider(&kubermaticv1.Datacenter{
		Spec: kubermaticv1.DatacenterSpec{
			AWS: &kubermaticv1.DatacenterSpecAWS{
				Region: os.Getenv(awsRegionEnvName),
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
	ctx := context.Background()

	defaultVPC, err := getDefaultVPC(ctx, provider.clientSet.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	defaultVPCID := *defaultVPC.VpcId

	// to properly test, we need the ID of a pre-existing security group
	sGroups, err := getSecurityGroupsWithClient(ctx, provider.clientSet.EC2)
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
			err := provider.ValidateCloudSpec(ctx, kubermaticv1.CloudSpec{AWS: testcase.cloudSpec})
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

	cluster, err := provider.InitializeCloudProvider(context.Background(), cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("InitializeCloudProvider should not have failed, but returned: %v", err)
	}

	if !kuberneteshelper.HasFinalizer(cluster, cleanupFinalizer) {
		t.Error("cluster should have cleanup finalizer, but does not")
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

func TestInitializeCloudProviderKeepsAnyData(t *testing.T) {
	provider := newCloudProvider(t)
	nope := "does-not-exist"

	// create a cluster with lots of broken data of which we expect
	// the InitializeCloudProvider() to NOT fix them.
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
		VPCID:               nope,
		SecurityGroupID:     nope,
		ControlPlaneRoleARN: nope,
		RouteTableID:        nope,
		InstanceProfileName: nope,
	})

	// As this is a somewhat synthetic usecase, we need to properly simulate that
	// the cluster was already reconciled, which includes having this finalizer.
	// Otherwise the code would try to tag the non-existing resources and fail.
	kuberneteshelper.AddFinalizer(cluster, cleanupFinalizer)

	cluster, err := provider.InitializeCloudProvider(context.Background(), cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("InitializeCloudProvider should not have failed, but returned: %v", err)
	}

	if cluster.Spec.Cloud.AWS.VPCID != nope {
		t.Error("cloud spec should have retained the VPC ID")
	}

	if cluster.Spec.Cloud.AWS.RouteTableID != nope {
		t.Error("cloud spec should have retained the route table ID")
	}

	if cluster.Spec.Cloud.AWS.SecurityGroupID != nope {
		t.Error("cloud spec should have retained the security group ID")
	}

	if cluster.Spec.Cloud.AWS.ControlPlaneRoleARN != nope {
		t.Error("cloud spec should have retained the control plane role name")
	}

	if cluster.Spec.Cloud.AWS.InstanceProfileName != nope {
		t.Error("cloud spec should have retained the instance profile name")
	}
}

func TestReconcileCluster(t *testing.T) {
	provider := newCloudProvider(t)
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})

	cluster, err := provider.ReconcileCluster(context.Background(), cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster should not have failed, but returned: %v", err)
	}

	if !kuberneteshelper.HasFinalizer(cluster, cleanupFinalizer) {
		t.Error("cluster should have cleanup finalizer, but does not")
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

func TestReconcileClusterFixesProblems(t *testing.T) {
	provider := newCloudProvider(t)
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
		SecurityGroupID: "does-not-exist",
	})

	cluster, err := provider.ReconcileCluster(context.Background(), cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster should not have failed, but returned: %v", err)
	}

	if cluster.Spec.Cloud.AWS.SecurityGroupID == "does-not-exist" {
		t.Error("cloud spec should have fixed the security group ID")
	}
}

func TestCleanUpCloudProvider(t *testing.T) {
	provider := newCloudProvider(t)
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{})

	// create a vanilla cluster
	cluster, err := provider.ReconcileCluster(context.Background(), cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster should not have failed, but returned: %v", err)
	}

	// quick sanity check
	if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		t.Error("cloud spec should have a security group ID")
	}

	// clean it up
	cluster, err = provider.CleanUpCloudProvider(context.Background(), cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("CleanUpCloudProvider should not have failed, but returned: %v", err)
	}

	// The actual cleanup logic is tested for each resource individually,
	// in this test we're interested in the provider's additional logic.

	// ensure no finalizer remains
	if kuberneteshelper.HasAnyFinalizer(cluster,
		cleanupFinalizer,
		securityGroupCleanupFinalizer,
		instanceProfileCleanupFinalizer,
		controlPlaneRoleCleanupFinalizer,
		tagCleanupFinalizer,
	) {
		t.Errorf("Cleaning up should have left no AWS finalizers on the cluster, but %v remained.", cluster.Finalizers)
	}
}

func TestIsValidRoleUpdate(t *testing.T) {
	testcases := []struct {
		oldValue string
		newValue string
		valid    bool
	}{
		// field is kept empty
		{
			oldValue: "",
			newValue: "",
			valid:    true,
		},
		// nothing was actually changed
		{
			oldValue: "oldstyle-value",
			newValue: "oldstyle-value",
			valid:    true,
		},
		// nothing is changed, but both are valid ARNs
		{
			oldValue: "arn:aws:iam::account:role/my-role",
			newValue: "arn:aws:iam::account:role/my-role",
			valid:    true,
		},
		// setting the value to an ARN for the first time is the happy path
		{
			oldValue: "",
			newValue: "arn:aws:iam::account:role/my-role",
			valid:    true,
		},
		// turning an old-style value into a real ARN is allowed (i.e. during upgrades)
		{
			oldValue: "my-role",
			newValue: "arn:aws:iam::account:role/my-role",
			valid:    true,
		},
		// do not allow to change the role during upgrades
		{
			oldValue: "my-role",
			newValue: "arn:aws:iam::account:role/my-other-role",
			valid:    false,
		},
		// changing ARNs is forbidden
		{
			oldValue: "arn:aws:iam::account:role/my-role",
			newValue: "arn:aws:iam::account:role/my-new-role",
			valid:    false,
		},
		// changing old-style values is forbidden because new values must be real ARNs
		{
			oldValue: "my-role",
			newValue: "my-new-role",
			valid:    false,
		},
		// cannot turn ARN into old-style value (even if it refers to the same role)
		{
			oldValue: "arn:aws:iam::account:role/my-role",
			newValue: "my-role",
			valid:    false,
		},
		// cannot remove the value once it's set
		{
			oldValue: "my-role",
			newValue: "",
			valid:    false,
		},
		// cannot remove the value once it's set
		{
			oldValue: "arn:aws:iam::account:role/my-role",
			newValue: "",
			valid:    false,
		},
	}

	for _, testcase := range testcases {
		t.Run(fmt.Sprintf("%s -> %s", testcase.oldValue, testcase.newValue), func(t *testing.T) {
			err := validateRoleUpdate(testcase.oldValue, testcase.newValue)
			if testcase.valid {
				if err != nil {
					t.Fatalf("Expected no error, but got: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("Expected an error, but got none.")
				}
			}
		})
	}
}
