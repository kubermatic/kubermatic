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

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
)

func ec2ClusterTag(clusterName string) ec2types.Tag {
	return ec2types.Tag{
		Key:   ptr.To(kubernetesClusterTagPrefix + clusterName),
		Value: ptr.To(""),
	}
}

func ec2OwnershipTag(clusterName string) ec2types.Tag {
	return ec2types.Tag{
		Key:   ptr.To(ownershipTagPrefix + clusterName),
		Value: ptr.To(""),
	}
}

func iamOwnershipTag(clusterName string) iamtypes.Tag {
	return iamtypes.Tag{
		Key:   ptr.To(ownershipTagPrefix + clusterName),
		Value: ptr.To(""),
	}
}

func reconcileClusterTags(ctx context.Context, client *ec2.Client, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	// tagging happens after a successful reconciliation, so we can rely on these fields not being empty
	resourceIDs := []string{
		cluster.Spec.Cloud.AWS.SecurityGroupID,
		cluster.Spec.Cloud.AWS.RouteTableID,
	}

	sOut, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{ec2VPCFilter(cluster.Spec.Cloud.AWS.VPCID)},
	})
	if err != nil {
		return cluster, fmt.Errorf("failed to list subnets: %w", err)
	}

	var subnetIDs []string
	for _, subnet := range sOut.Subnets {
		resourceIDs = append(resourceIDs, ptr.Deref(subnet.SubnetId, ""))
		subnetIDs = append(subnetIDs, ptr.Deref(subnet.SubnetId, ""))
	}

	_, err = client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: resourceIDs,
		Tags:      []ec2types.Tag{ec2ClusterTag(cluster.Name)},
	})
	if err != nil {
		return cluster, fmt.Errorf("failed to tag resources (one of securityGroup (%s), routeTable (%s) and/or subnets (%v)): %w",
			cluster.Spec.Cloud.AWS.SecurityGroupID, cluster.Spec.Cloud.AWS.RouteTableID, subnetIDs, err)
	}

	return cluster, nil
}

func cleanUpTags(ctx context.Context, client *ec2.Client, cluster *kubermaticv1.Cluster) error {
	// Instead of trying to keep track of all the things we might have
	// tagged, we instead simply list all tagged resources. It's a few
	// more API calls, but saves us a lot of trouble in the code w.r.t.
	// resource IDs.
	// We always want to try our very best to clean up after ourselves.
	// This means that even if the security group ID is already missing
	// on the cluster object, we still do not want to accidentally orphan
	// it.
	// Ownership tags do not need to be deleted, as they disappear when their
	// objects are deleted.

	tag := ec2ClusterTag(cluster.Name)

	resourceIDs := []string{}
	filters := []ec2types.Filter{
		ec2VPCFilter(cluster.Spec.Cloud.AWS.VPCID),
		{
			Name:   ptr.To("tag-key"),
			Values: []string{ptr.Deref(tag.Key, "")},
		},
	}

	// list subnets (we do not create subnets, but we tagged all of them
	// to make the AWS CCM LoadBalancer integration work)
	subnets, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{Filters: filters})
	if err != nil {
		return fmt.Errorf("failed to list subnets: %w", err)
	}

	for _, subnet := range subnets.Subnets {
		resourceIDs = append(resourceIDs, ptr.Deref(subnet.SubnetId, ""))
	}

	// list security groups
	securityGroups, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{Filters: filters})
	if err != nil {
		return fmt.Errorf("failed to list security groups: %w", err)
	}

	for _, group := range securityGroups.SecurityGroups {
		resourceIDs = append(resourceIDs, ptr.Deref(group.GroupId, ""))
	}

	// list route tables
	routeTables, err := client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{Filters: filters})
	if err != nil {
		return fmt.Errorf("failed to list route tables: %w", err)
	}

	for _, rt := range routeTables.RouteTables {
		resourceIDs = append(resourceIDs, ptr.Deref(rt.RouteTableId, ""))
	}

	// remove tag
	if len(resourceIDs) > 0 {
		_, err = client.DeleteTags(ctx, &ec2.DeleteTagsInput{
			Resources: resourceIDs,
			Tags:      []ec2types.Tag{tag},
		})
	}

	return err
}
