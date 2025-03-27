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

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func assertTagExistence(t *testing.T, ctx context.Context, client *ec2.Client, cluster *kubermaticv1.Cluster, vpc *ec2types.Vpc, expected bool) {
	rt, err := getDefaultRouteTable(ctx, client, *vpc.VpcId)
	if err != nil {
		t.Fatalf("getDefaultRouteTable should not have errored, but returned %v", err)
	}

	if hasEC2Tag(ec2ClusterTag(cluster.Name), rt.Tags) != expected {
		if expected {
			t.Errorf("route table %q should have cluster tag, but does not", *rt.RouteTableId)
		} else {
			t.Errorf("route table %q should not have cluster tag, but does", *rt.RouteTableId)
		}
	}

	securityGroupID := cluster.Spec.Cloud.AWS.SecurityGroupID

	group, err := getSecurityGroupByID(ctx, client, vpc, securityGroupID)
	if err != nil {
		t.Fatalf("getSecurityGroupByID should not have errored, but returned %v", err)
	}

	if hasEC2Tag(ec2ClusterTag(cluster.Name), group.Tags) != expected {
		if expected {
			t.Errorf("security group %q should have cluster tag, but does not", securityGroupID)
		} else {
			t.Errorf("security group %q should not have cluster tag, but does", securityGroupID)
		}
	}

	subnets, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{ec2VPCFilter(cluster.Spec.Cloud.AWS.VPCID)},
	})
	if err != nil {
		t.Fatalf("DescribeSubnets should not have errored, but returned %v", err)
	}

	for _, subnet := range subnets.Subnets {
		if hasEC2Tag(ec2ClusterTag(cluster.Name), group.Tags) != expected {
			if expected {
				t.Errorf("subnet %q should have cluster tag, but does not", *subnet.SubnetId)
			} else {
				t.Errorf("subnet %q should not have cluster tag, but does", *subnet.SubnetId)
			}
		}
	}
}

func TestReconcileClusterTags(t *testing.T) {
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

	cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
		VPCID:           defaultVPCID,
		RouteTableID:    defaultRouteTableID,
		SecurityGroupID: securityGroupID,
	})

	cluster, err = reconcileClusterTags(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("reconcileClusterTags should not have errored, but returned %v", err)
	}

	assertTagExistence(t, ctx, cs.EC2, cluster, defaultVPC, true)

	// reconciling again should not do anything, and also not fail
	_, err = reconcileClusterTags(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
	if err != nil {
		t.Fatalf("reconcileClusterTags (2) should not have errored, but returned %v", err)
	}

	assertTagExistence(t, ctx, cs.EC2, cluster, defaultVPC, true)
}

func TestCleanUpTags(t *testing.T) {
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

	t.Run("vanilla-case", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:           defaultVPCID,
			RouteTableID:    defaultRouteTableID,
			SecurityGroupID: securityGroupID,
		})

		// create resources and tag them
		cluster, err = reconcileClusterTags(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileClusterTags should not have errored, but returned %v", err)
		}

		// ensure all tags are set
		assertTagExistence(t, ctx, cs.EC2, cluster, defaultVPC, true)

		// and now get rid of them again
		if err = cleanUpTags(ctx, cs.EC2, cluster); err != nil {
			t.Fatalf("cleanUpTags should not have errored, but returned %v", err)
		}

		// ensure all tags are gone
		assertTagExistence(t, ctx, cs.EC2, cluster, defaultVPC, false)
	})

	t.Run("recover-and-untag-resources", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AWSCloudSpec{
			VPCID:           defaultVPCID,
			RouteTableID:    defaultRouteTableID,
			SecurityGroupID: securityGroupID,
		})

		// create resources and tag them
		cluster, err = reconcileClusterTags(ctx, cs.EC2, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("reconcileClusterTags should not have errored, but returned %v", err)
		}

		// ensure all tags are set
		assertTagExistence(t, ctx, cs.EC2, cluster, defaultVPC, true)

		// demonstrate that deleting tags works even when the cluster object is broken
		backup := cluster.DeepCopy()

		cluster.Spec.Cloud.AWS.RouteTableID = ""
		cluster.Spec.Cloud.AWS.SecurityGroupID = ""

		// and now get rid of them again
		if err = cleanUpTags(ctx, cs.EC2, cluster); err != nil {
			t.Fatalf("cleanUpTags should not have errored, but returned %v", err)
		}

		// ensure all tags are gone
		assertTagExistence(t, ctx, cs.EC2, backup, defaultVPC, false)
	})
}
