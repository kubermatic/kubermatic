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

package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

func ec2ClusterTag(clusterName string) *ec2.Tag {
	return &ec2.Tag{
		Key:   aws.String(tagNameKubernetesClusterPrefix + clusterName),
		Value: aws.String(""),
	}
}

func ec2OwnershipTag(clusterName string) *ec2.Tag {
	return &ec2.Tag{
		Key:   aws.String(ownershipTagPrefix + clusterName),
		Value: aws.String(""),
	}
}

func iamClusterTag(clusterName string) *iam.Tag {
	return &iam.Tag{
		Key:   aws.String(tagNameKubernetesClusterPrefix + clusterName),
		Value: aws.String(""),
	}
}

func iamOwnershipTag(clusterName string) *iam.Tag {
	return &iam.Tag{
		Key:   aws.String(ownershipTagPrefix + clusterName),
		Value: aws.String(""),
	}
}

func tagResources(client *ClientSet, cluster *kubermaticv1.Cluster) error {
	resourceIDs := []*string{
		&cluster.Spec.Cloud.AWS.SecurityGroupID,
		&cluster.Spec.Cloud.AWS.RouteTableID,
	}

	sOut, err := client.EC2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{{
			Name:   aws.String("vpc-id"),
			Values: aws.StringSlice([]string{cluster.Spec.Cloud.AWS.VPCID}),
		}},
	})
	if err != nil {
		return fmt.Errorf("failed to list subnets: %w", err)
	}

	var subnetIDs []string
	for _, subnet := range sOut.Subnets {
		resourceIDs = append(resourceIDs, subnet.SubnetId)
		subnetIDs = append(subnetIDs, *subnet.SubnetId)
	}

	_, err = client.EC2.CreateTags(&ec2.CreateTagsInput{
		Resources: resourceIDs,
		Tags:      []*ec2.Tag{ec2ClusterTag(cluster.Name)},
	})
	if err != nil {
		return fmt.Errorf("failed to tag resources (one of securityGroup (%s), routeTable (%s) and/or subnets (%v)): %w",
			cluster.Spec.Cloud.AWS.SecurityGroupID, cluster.Spec.Cloud.AWS.RouteTableID, subnetIDs, err)
	}

	// TODO: tag IAM resources

	return nil
}

func deleteTags(client *ClientSet, cluster *kubermaticv1.Cluster) error {
	// Instead of trying to keep track of all the things we might have
	// tagged, we instead simply list all tagged resources. It's a few
	// more API calls, but saves us a lot of trouble in the code w.r.t.
	// resource IDs.
	// We always want to try our very best to clean up after ourselves.
	// This means that even if the security group ID is already missing
	// on the cluster object, we still do not want to accidentally orphan
	// it.
	// Both the EC2 and IAM functions implement this "catch all" behaviour.

	if err := deleteEC2Tags(client, cluster); err != nil {
		return fmt.Errorf("failed to delete EC2 tags: %w", err)
	}

	// if err := deleteIAMTags(client, cluster); err != nil {
	// 	return fmt.Errorf("failed to delete IAM tags: %w", err)
	// }

	return nil
}

func deleteEC2Tags(client *ClientSet, cluster *kubermaticv1.Cluster) error {
	tag := ec2ClusterTag(cluster.Name)

	resourceIDs := []*string{}
	filters := []*ec2.Filter{{
		Name:   aws.String("vpc-id"),
		Values: aws.StringSlice([]string{cluster.Spec.Cloud.AWS.VPCID}),
	}, {
		Name:   aws.String("tag-key"),
		Values: aws.StringSlice([]string{*tag.Key}),
	}}

	// list subnets (we do not create subnets, but we tagged all of them
	// to make the AWS CCM LoadBalancer integration work)
	subnets, err := client.EC2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("failed to list subnets: %w", err)
	}

	for _, subnet := range subnets.Subnets {
		resourceIDs = append(resourceIDs, subnet.SubnetId)
	}

	// list security groups
	securityGroups, err := client.EC2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("failed to list security groups: %w", err)
	}

	for _, group := range securityGroups.SecurityGroups {
		resourceIDs = append(resourceIDs, group.GroupId)
	}

	// list route tables
	routeTables, err := client.EC2.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("failed to list route tables: %w", err)
	}

	for _, rt := range routeTables.RouteTables {
		resourceIDs = append(resourceIDs, rt.RouteTableId)
	}

	// remove tags
	_, err = client.EC2.DeleteTags(&ec2.DeleteTagsInput{
		Resources: resourceIDs,
		Tags:      []*ec2.Tag{tag},
	})

	return err
}

// func deleteIAMTags(client *ClientSet, cluster *kubermaticv1.Cluster) error {
// 	tag := iamClusterTag(cluster.Name)

// 	resourceIDs := []*string{}
// 	filters := []*ec2.Filter{{
// 		Name:   aws.String("vpc-id"),
// 		Values: aws.StringSlice([]string{cluster.Spec.Cloud.AWS.VPCID}),
// 	}, {
// 		Name:   aws.String("tag-key"),
// 		Values: aws.StringSlice([]string{*tag.Key}),
// 	}}

// 	// list instance profiles
// 	subnets, err := client.EC2.DescribeSubnets(&ec2.DescribeSubnetsInput{
// 		Filters: filters,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to list subnets: %w", err)
// 	}

// 	for _, subnet := range subnets.Subnets {
// 		resourceIDs = append(resourceIDs, subnet.SubnetId)
// 	}

// 	// list security groups
// 	securityGroups, err := client.EC2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
// 		Filters: filters,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to list security groups: %w", err)
// 	}

// 	for _, group := range securityGroups.SecurityGroups {
// 		resourceIDs = append(resourceIDs, group.GroupId)
// 	}

// 	// list route tables
// 	routeTables, err := client.EC2.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
// 		Filters: filters,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to list route tables: %w", err)
// 	}

// 	for _, rt := range routeTables.RouteTables {
// 		resourceIDs = append(resourceIDs, rt.RouteTableId)
// 	}

// 	// remove tags
// 	_, err = client.EC2.DeleteTags(&ec2.DeleteTagsInput{
// 		Resources: resourceIDs,
// 		Tags:      []*ec2.Tag{tag},
// 	})

// 	return err
// }
