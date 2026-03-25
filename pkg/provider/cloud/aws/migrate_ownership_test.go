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
	"testing"

	iam "github.com/aws/aws-sdk-go-v2/service/iam"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
)

func TestBackfillOwnershipTags(t *testing.T) {
	provider := newCloudProvider(t)
	finalizer := tagCleanupFinalizer

	// create a vanilla cluster
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
		AccessKeyID:     nope,
		SecretAccessKey: nope,
	})
	kuberneteshelper.AddFinalizer(cluster, finalizer)

	cluster, err := provider.ReconcileCluster(context.Background(), cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster should not have failed, but returned: %v", err)
	}

	// the finalizer should be gone
	if kuberneteshelper.HasFinalizer(cluster, finalizer) {
		t.Fatalf("Reconciling should have removed the %q finalizer", finalizer)
	}
}

// All three tests below follow the same pattern:
// 1. reconcile a cluster that uses pre-existing resources
// 2. ensure no tags and no finalizers are set
// 3. add the finalizer
// 4. reconcile, i.e. backfill, again
// 5. ensure owner tag is set and finalizer is gone
// 6. add the finalizer again
// 7. reconcile, i.e. backfill, again to see if "double tagging" does not cause problems

func TestBackfillOwnershipTagsAdoptsSecurityGroup(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)
	provider := newCloudProvider(t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	// to properly test, we need the ID of a pre-existing security group
	sGroups, err := getSecurityGroupsWithClient(ctx, provider.clientSet.EC2)
	if err != nil {
		t.Fatalf("getSecurityGroupsWithClient should not have errored, but returned %v", err)
	}
	if len(sGroups) == 0 {
		t.Fatal("getSecurityGroupsWithClient should have found at least one security group")
	}

	securityGroupID := *sGroups[0].GroupId
	finalizer := securityGroupCleanupFinalizer

	// create a vanilla cluster that uses an existing SG;
	// this will not put an owner tag on the SG
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
		SecurityGroupID: securityGroupID,
		AccessKeyID:     nope,
		SecretAccessKey: nope,
	})
	cluster, err = provider.ReconcileCluster(ctx, cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster should not have failed, but returned: %v", err)
	}

	ownerTag := ec2OwnershipTag(cluster.Name)

	// assert that there really is no owner tag
	group, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, securityGroupID)
	if err != nil {
		t.Fatalf("Failed to get security group: %v", err)
	}

	if hasEC2Tag(ownerTag, group.Tags) {
		t.Fatalf("Security group should not have received an owner tag, but did.")
	}

	// ... and no finalizer
	if kuberneteshelper.HasFinalizer(cluster, finalizer) {
		t.Fatalf("Reconciling should never add the legacy %q finalizer, but it did.", finalizer)
	}

	// now add the finalizer and thereby signify that a backfilling needs to happen
	kuberneteshelper.AddFinalizer(cluster, finalizer)

	// migrate!
	cluster, err = provider.ReconcileCluster(ctx, cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster (2) should not have failed, but returned: %v", err)
	}

	// finalizer should be gone
	if kuberneteshelper.HasFinalizer(cluster, finalizer) {
		t.Fatalf("Backfilling should have removed the %q finalizer, but did not.", finalizer)
	}

	// and an owner tag should have appeared
	group, err = getSecurityGroupByID(ctx, cs.EC2, defaultVPC, securityGroupID)
	if err != nil {
		t.Fatalf("Failed to get security group: %v", err)
	}

	if !hasEC2Tag(ownerTag, group.Tags) {
		t.Fatalf("Security group should have received an owner tag, but did not.")
	}

	// pretend that we haven't backfilled yet, so we can see what happens if we try to add the owner tag again
	kuberneteshelper.AddFinalizer(cluster, finalizer)

	// This should be a NOP.
	if _, err = provider.ReconcileCluster(ctx, cluster, testClusterUpdater(cluster)); err != nil {
		t.Fatalf("ReconcileCluster (3) should not have failed, but returned: %v", err)
	}
}

func TestBackfillOwnershipTagsAdoptsInstanceProfile(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)
	provider := newCloudProvider(t)

	profileName := "adopt-me-" + rand.String(10)
	finalizer := instanceProfileCleanupFinalizer

	// create an instance profile
	createProfileInput := &iam.CreateInstanceProfileInput{
		InstanceProfileName: ptr.To(profileName),
	}

	if _, err := cs.IAM.CreateInstanceProfile(ctx, createProfileInput); err != nil {
		t.Fatalf("Failed to create dummy instance profile: %v", err)
	}

	// this will not put an owner tag on the profile
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
		InstanceProfileName: profileName,
		AccessKeyID:         nope,
		SecretAccessKey:     nope,
	})
	cluster, err := provider.ReconcileCluster(ctx, cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster should not have failed, but returned: %v", err)
	}

	ownerTag := iamOwnershipTag(cluster.Name)

	// assert that there really is no owner tag
	profile, err := getInstanceProfile(ctx, cs.IAM, profileName)
	if err != nil {
		t.Fatalf("Failed to get instance profile: %v", err)
	}

	if hasIAMTag(ownerTag, profile.Tags) {
		t.Fatalf("Instance profile should not have received an owner tag, but did.")
	}

	// ... and no finalizer
	if kuberneteshelper.HasFinalizer(cluster, finalizer) {
		t.Fatalf("Reconciling should never add the legacy %q finalizer, but it did.", finalizer)
	}

	// now add the finalizer and thereby signify that a backfilling needs to happen
	kuberneteshelper.AddFinalizer(cluster, finalizer)

	// migrate!
	cluster, err = provider.ReconcileCluster(ctx, cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster (2) should not have failed, but returned: %v", err)
	}

	// finalizer should be gone
	if kuberneteshelper.HasFinalizer(cluster, finalizer) {
		t.Fatalf("Backfilling should have removed the %q finalizer, but did not.", finalizer)
	}

	// and an owner tag should have appeared
	profile, err = getInstanceProfile(ctx, cs.IAM, profileName)
	if err != nil {
		t.Fatalf("Failed to get instance profile: %v", err)
	}

	if !hasIAMTag(ownerTag, profile.Tags) {
		t.Fatalf("Instance profile should have received an owner tag, but did not.")
	}

	// pretend that we haven't backfilled yet, so we can see what happens if we try to add the owner tag again
	kuberneteshelper.AddFinalizer(cluster, finalizer)

	// This should be a NOP.
	if _, err = provider.ReconcileCluster(ctx, cluster, testClusterUpdater(cluster)); err != nil {
		t.Fatalf("ReconcileCluster (3) should not have failed, but returned: %v", err)
	}
}

func TestBackfillOwnershipTagsAdoptsControlPlaneRole(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)
	provider := newCloudProvider(t)

	roleName := "adopt-me-" + rand.String(10)
	finalizer := controlPlaneRoleCleanupFinalizer

	// create a role
	createRoleInput := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: ptr.To(assumeRolePolicy),
		RoleName:                 ptr.To(roleName),
	}

	output, err := cs.IAM.CreateRole(ctx, createRoleInput)
	if err != nil {
		t.Fatalf("Failed to create dummy role: %v", err)
	}

	roleARN := *output.Role.Arn

	// this will not put an owner tag on the role
	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
		ControlPlaneRoleARN: roleARN,
		AccessKeyID:         nope,
		SecretAccessKey:     nope,
	})

	cluster, err = provider.ReconcileCluster(ctx, cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster should not have failed, but returned: %v", err)
	}

	ownerTag := iamOwnershipTag(cluster.Name)

	// assert that there really is no owner tag
	role, err := getRole(ctx, cs.IAM, roleName)
	if err != nil {
		t.Fatalf("Failed to get role: %v", err)
	}

	if hasIAMTag(ownerTag, role.Tags) {
		t.Fatalf("role should not have received an owner tag, but did.")
	}

	// ... and no finalizer
	if kuberneteshelper.HasFinalizer(cluster, finalizer) {
		t.Fatalf("Reconciling should never add the legacy %q finalizer, but it did.", finalizer)
	}

	// now add the finalizer and thereby signify that a backfilling needs to happen
	kuberneteshelper.AddFinalizer(cluster, finalizer)

	// migrate!
	cluster, err = provider.ReconcileCluster(ctx, cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("ReconcileCluster (2) should not have failed, but returned: %v", err)
	}

	// finalizer should be gone
	if kuberneteshelper.HasFinalizer(cluster, finalizer) {
		t.Fatalf("Backfilling should have removed the %q finalizer, but did not.", finalizer)
	}

	// and an owner tag should have appeared
	role, err = getRole(ctx, cs.IAM, roleName)
	if err != nil {
		t.Fatalf("Failed to get role: %v", err)
	}

	if !hasIAMTag(ownerTag, role.Tags) {
		t.Fatalf("role should have received an owner tag, but did not.")
	}

	// pretend that we haven't backfilled yet, so we can see what happens if we try to add the owner tag again
	kuberneteshelper.AddFinalizer(cluster, finalizer)

	// This should be a NOP.
	if _, err = provider.ReconcileCluster(ctx, cluster, testClusterUpdater(cluster)); err != nil {
		t.Fatalf("ReconcileCluster (3) should not have failed, but returned: %v", err)
	}
}
