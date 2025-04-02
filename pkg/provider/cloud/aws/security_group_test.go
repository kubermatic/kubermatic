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

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/utils/ptr"
)

func TestGetSecurityGroupByID(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	t.Run("invalid-vpc-invalid-sg", func(t *testing.T) {
		if _, err := getSecurityGroupByID(ctx, cs.EC2, nil, "does-not-exist"); err == nil {
			t.Fatalf("getSecurityGroupByID should have errored, but returned %v", err)
		}
	})

	t.Run("valid-vpc-invalid-sg", func(t *testing.T) {
		if _, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, "does-not-exist"); err == nil {
			t.Fatalf("getSecurityGroupByID should have errored, but returned %v", err)
		}
	})
}

func assertSecurityGroup(t *testing.T, cluster *kubermaticv1.Cluster, group *ec2types.SecurityGroup, expectOwnerTag bool) {
	if group.GroupName == nil || *group.GroupName == "" {
		t.Error("security group should have a name, but its empty")
	}

	if group.Description == nil || *group.Description == "" {
		t.Error("security group should have a description, but its empty")
	}

	if expectOwnerTag {
		if len(group.Tags) == 0 {
			t.Error("security group should have tags, but has none")
		} else {
			ownerTag := ec2OwnershipTag(cluster.Name)
			exists := false

			for _, tag := range group.Tags {
				if *tag.Key == *ownerTag.Key {
					exists = true
				}
			}

			if !exists {
				t.Errorf("security group should have %q tag, but was not found", *ownerTag.Key)
			}
		}
	}

	permissions := getCommonSecurityGroupPermissions(*group.GroupId, true, true)

	lowPort, highPort := getNodePortRange(cluster)
	permissions = append(permissions, getNodePortSecurityGroupPermissions(lowPort, highPort, []string{resources.IPv4MatchAnyCIDR}, []string{resources.IPv6MatchAnyCIDR})...)

	missingPermissions := []ec2types.IpPermission{}

	for _, perm := range permissions {
		found := false

		for _, expectedPerm := range group.IpPermissions {
			// normalize data and remove data inserted by AWS
			for i := range expectedPerm.UserIdGroupPairs {
				expectedPerm.UserIdGroupPairs[i].UserId = nil
			}

			if compareIPPermissions(perm, expectedPerm) {
				found = true
				break
			}
		}

		if !found {
			missingPermissions = append(missingPermissions, perm)
		}
	}

	if len(missingPermissions) > 0 {
		t.Errorf("security group is missing the following IP permissions: %+v", missingPermissions)
	}
}

func compareIPPermissions(perm1 ec2types.IpPermission, perm2 ec2types.IpPermission) bool {
	if ptr.Deref[int32](perm1.FromPort, -1) != ptr.Deref[int32](perm2.FromPort, -1) {
		return false
	}

	if ptr.Deref[int32](perm1.ToPort, -1) != ptr.Deref[int32](perm2.ToPort, -1) {
		return false
	}

	if ptr.Deref(perm1.IpProtocol, "") != ptr.Deref(perm2.IpProtocol, "") {
		return false
	}

	if len(perm1.IpRanges) != len(perm2.IpRanges) {
		return false
	}

	for i := range perm1.IpRanges {
		if ptr.Deref(perm1.IpRanges[i].CidrIp, "") != ptr.Deref(perm2.IpRanges[i].CidrIp, "") {
			return false
		}
	}

	if len(perm1.Ipv6Ranges) != len(perm2.Ipv6Ranges) {
		return false
	}

	for i := range perm1.Ipv6Ranges {
		if ptr.Deref(perm1.Ipv6Ranges[i].CidrIpv6, "") != ptr.Deref(perm2.Ipv6Ranges[i].CidrIpv6, "") {
			return false
		}
	}

	if len(perm1.UserIdGroupPairs) != len(perm2.UserIdGroupPairs) {
		return false
	}

	for i := range perm1.UserIdGroupPairs {
		if ptr.Deref(perm1.UserIdGroupPairs[i].GroupId, "") != ptr.Deref(perm2.UserIdGroupPairs[i].GroupId, "") {
			return false
		}

		if ptr.Deref(perm1.UserIdGroupPairs[i].GroupName, "") != ptr.Deref(perm2.UserIdGroupPairs[i].GroupName, "") {
			return false
		}

		if ptr.Deref(perm1.UserIdGroupPairs[i].PeeringStatus, "") != ptr.Deref(perm2.UserIdGroupPairs[i].PeeringStatus, "") {
			return false
		}

		if ptr.Deref(perm1.UserIdGroupPairs[i].VpcId, "") != ptr.Deref(perm2.UserIdGroupPairs[i].VpcId, "") {
			return false
		}

		if ptr.Deref(perm1.UserIdGroupPairs[i].VpcPeeringConnectionId, "") != ptr.Deref(perm2.UserIdGroupPairs[i].VpcPeeringConnectionId, "") {
			return false
		}
	}

	if len(perm1.PrefixListIds) != len(perm2.PrefixListIds) {
		return false
	}

	for i := range perm1.PrefixListIds {
		if ptr.Deref(perm1.PrefixListIds[i].PrefixListId, "") != ptr.Deref(perm2.PrefixListIds[i].PrefixListId, "") {
			return false
		}
	}

	return true
}

func TestReconcileSecurityGroup(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	defaultRT, err := getDefaultRouteTable(ctx, cs.EC2, *defaultVPC.VpcId)
	if err != nil {
		t.Fatalf("getDefaultRouteTable should not have errored, but returned %v", err)
	}

	// to properly test, we need the ID of a pre-existing security group
	sGroups, err := getSecurityGroupsWithClient(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getSecurityGroupsWithClient should not have errored, but returned %v", err)
	}
	if len(sGroups) == 0 {
		t.Fatal("getSecurityGroupsWithClient should have found at least one security group")
	}

	defaultVPCID := *defaultVPC.VpcId
	defaultRouteTableID := *defaultRT.RouteTableId
	securityGroupID := *sGroups[0].GroupId

	t.Run("everything-is-fine", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:           defaultVPCID,
			RouteTableID:    defaultRouteTableID,
			SecurityGroupID: securityGroupID,
		})

		cluster, err = reconcileSecurityGroup(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileSecurityGroup should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.VPCID != defaultVPCID {
			t.Errorf("cloud spec should have retained VPC ID %q, but is now %q", defaultVPCID, cluster.Spec.Cloud.AWS.VPCID)
		}

		if cluster.Spec.Cloud.AWS.RouteTableID != defaultRouteTableID {
			t.Errorf("cloud spec should have retained route table ID %q, but is now %q", defaultRouteTableID, cluster.Spec.Cloud.AWS.RouteTableID)
		}

		if cluster.Spec.Cloud.AWS.SecurityGroupID != securityGroupID {
			t.Errorf("cloud spec should have retained security group ID %q, but is now %q", securityGroupID, cluster.Spec.Cloud.AWS.SecurityGroupID)
		}

		group, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, cluster.Spec.Cloud.AWS.SecurityGroupID)
		if err != nil {
			t.Fatalf("getSecurityGroupByID should have not errored, but returned %v", err)
		}

		// do not assert an ownership tag here, because a valid SG ID was given
		assertSecurityGroup(t, cluster, group, false)
	})

	t.Run("no-security-group-yet", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:        defaultVPCID,
			RouteTableID: defaultRouteTableID,
		})

		cluster, err = reconcileSecurityGroup(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileSecurityGroup should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.VPCID != defaultVPCID {
			t.Errorf("cloud spec should have retained VPC ID %q, but is now %q", defaultVPCID, cluster.Spec.Cloud.AWS.VPCID)
		}

		if cluster.Spec.Cloud.AWS.RouteTableID != defaultRouteTableID {
			t.Errorf("cloud spec should have found route table ID %q, but is now %q", defaultRouteTableID, cluster.Spec.Cloud.AWS.RouteTableID)
		}

		if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
			t.Fatalf("cloud spec should have created a security group and stored its ID, but the field is empty")
		}

		group, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, cluster.Spec.Cloud.AWS.SecurityGroupID)
		if err != nil {
			t.Fatalf("getSecurityGroupByID should have not errored, but returned %v", err)
		}

		assertSecurityGroup(t, cluster, group, true)
	})

	t.Run("invalid-security-group", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:           defaultVPCID,
			RouteTableID:    defaultRouteTableID,
			SecurityGroupID: "does-not-exist",
		})

		cluster, err = reconcileSecurityGroup(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileSecurityGroup should not have errored, but returned %v", err)
		}

		if cluster.Spec.Cloud.AWS.VPCID != defaultVPCID {
			t.Errorf("cloud spec should have retained VPC ID %q, but is now %q", defaultVPCID, cluster.Spec.Cloud.AWS.VPCID)
		}

		if cluster.Spec.Cloud.AWS.RouteTableID != defaultRouteTableID {
			t.Errorf("cloud spec should have found route table ID %q, but is now %q", defaultRouteTableID, cluster.Spec.Cloud.AWS.RouteTableID)
		}

		if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
			t.Fatalf("cloud spec should have created a security group and stored its ID, but the field is empty")
		}

		group, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, cluster.Spec.Cloud.AWS.SecurityGroupID)
		if err != nil {
			t.Fatalf("getSecurityGroupByID should have not errored, but returned %v", err)
		}

		assertSecurityGroup(t, cluster, group, true)
	})

	t.Run("find-our-self-created-security-group", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:        defaultVPCID,
			RouteTableID: defaultRouteTableID,
		})

		// reconcile once to create a security group
		cluster, err = reconcileSecurityGroup(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileSecurityGroup should not have errored, but returned %v", err)
		}

		groupID := cluster.Spec.Cloud.AWS.SecurityGroupID

		if groupID == "" {
			t.Fatalf("cloud spec should have created a security group and stored its ID, but the field is empty")
		}

		group, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, cluster.Spec.Cloud.AWS.SecurityGroupID)
		if err != nil {
			t.Fatalf("getSecurityGroupByID should have not errored, but returned %v", err)
		}

		assertSecurityGroup(t, cluster, group, true)

		// reconcile again to see if we find the group
		cluster.Spec.Cloud.AWS.SecurityGroupID = ""

		cluster, err = reconcileSecurityGroup(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileSecurityGroup should not have errored, but returned %v", err)
		}

		if groupID != cluster.Spec.Cloud.AWS.SecurityGroupID {
			t.Fatalf("cloud spec should have created a found the group we created earlier (%s), but created another one (%s)", groupID, cluster.Spec.Cloud.AWS.SecurityGroupID)
		}

		assertSecurityGroup(t, cluster, group, true)
	})
}

func TestCleanUpSecurityGroup(t *testing.T) {
	ctx := context.Background()
	cs := getTestClientSet(ctx, t)

	defaultVPC, err := getDefaultVPC(ctx, cs.EC2)
	if err != nil {
		t.Fatalf("getDefaultVPC should not have errored, but returned %v", err)
	}

	defaultVPCID := *defaultVPC.VpcId

	t.Run("everything-is-fine", func(t *testing.T) {
		// reconcile once to create a new SG with ownership tag
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: defaultVPCID,
		})

		cluster, err = reconcileSecurityGroup(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileSecurityGroup should not have errored, but returned %v", err)
		}

		// assert that the group exists now
		if _, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, cluster.Spec.Cloud.AWS.SecurityGroupID); err != nil {
			t.Fatalf("getSecurityGroupByID should have not errored, but returned %v", err)
		}

		// and now get rid of it again
		if err = cleanUpSecurityGroup(ctx, cs.EC2, cluster); err != nil {
			t.Fatalf("cleanUpSecurityGroup should not have errored, but returned %v", err)
		}

		// assert that the group is gone
		if _, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, cluster.Spec.Cloud.AWS.SecurityGroupID); err == nil {
			t.Fatal("getSecurityGroupByID should have errored, but did not")
		}
	})

	t.Run("group-exists-but-id-is-missing", func(t *testing.T) {
		// reconcile once to create a new SG with ownership tag
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: defaultVPCID,
		})

		cluster, err = reconcileSecurityGroup(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileSecurityGroup should not have errored, but returned %v", err)
		}

		// assert that the group exists now
		if _, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, cluster.Spec.Cloud.AWS.SecurityGroupID); err != nil {
			t.Fatalf("getSecurityGroupByID should have not errored, but returned %v", err)
		}

		// break the cluster
		cluster.Spec.Cloud.AWS.SecurityGroupID = ""

		// and now get rid of it again
		if err = cleanUpSecurityGroup(ctx, cs.EC2, cluster); err != nil {
			t.Fatalf("cleanUpSecurityGroup should not have errored, but returned %v", err)
		}

		// assert that the group is gone
		if _, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, cluster.Spec.Cloud.AWS.SecurityGroupID); err == nil {
			t.Fatal("getSecurityGroupByID should have errored, but did not")
		}
	})

	t.Run("already-cleaned", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID: defaultVPCID,
		})

		// this should not do anything
		if err = cleanUpSecurityGroup(ctx, cs.EC2, cluster); err != nil {
			t.Fatalf("cleanUpSecurityGroup should not have errored, but returned %v", err)
		}
	})

	t.Run("bogus-security-group-id", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:           defaultVPCID,
			SecurityGroupID: "does-not-exist",
		})

		// this should not do anything
		if err = cleanUpSecurityGroup(ctx, cs.EC2, cluster); err != nil {
			t.Fatalf("cleanUpSecurityGroup should not have errored, but returned %v", err)
		}
	})

	t.Run("use-foreign-group-that-must-not-be-deleted", func(t *testing.T) {
		// reconcile a dummy cluster to create a security group
		dummyCluster := makeCluster(&kubermaticv1.AWSCloudSpec{VPCID: defaultVPCID})

		dummyCluster, err = reconcileSecurityGroup(ctx, cs.EC2, dummyCluster, testClusterUpdater(dummyCluster))
		if err != nil {
			t.Fatalf("reconcileSecurityGroup should not have errored, but returned %v", err)
		}

		// assert that the group exists now
		if _, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, dummyCluster.Spec.Cloud.AWS.SecurityGroupID); err != nil {
			t.Fatalf("getSecurityGroupByID should have not errored, but returned %v", err)
		}

		// and now use the dummyCluster's SG for another cluster, which will not own the SG.
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:           defaultVPCID,
			SecurityGroupID: dummyCluster.Spec.Cloud.AWS.SecurityGroupID,
		})

		// clean up
		if err = cleanUpSecurityGroup(ctx, cs.EC2, cluster); err != nil {
			t.Fatalf("cleanUpSecurityGroup should not have errored, but returned %v", err)
		}

		// assert that the group still exists
		if _, err := getSecurityGroupByID(ctx, cs.EC2, defaultVPC, dummyCluster.Spec.Cloud.AWS.SecurityGroupID); err != nil {
			t.Fatal("getSecurityGroupByID should have remained, but was removed")
		}
	})
}
