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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/iam"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
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

func iamOwnershipTag(clusterName string) *iam.Tag {
	return &iam.Tag{
		Key:   aws.String(ownershipTagPrefix + clusterName),
		Value: aws.String(""),
	}
}

func reconcileClusterTags(client ec2iface.EC2API, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	resourceIDs := []*string{
		&cluster.Spec.Cloud.AWS.SecurityGroupID,
		&cluster.Spec.Cloud.AWS.RouteTableID,
	}

	sOut, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{ec2VPCFilter(cluster.Spec.Cloud.AWS.VPCID)},
	})
	if err != nil {
		return cluster, fmt.Errorf("failed to list subnets: %w", err)
	}

	var subnetIDs []string
	for _, subnet := range sOut.Subnets {
		resourceIDs = append(resourceIDs, subnet.SubnetId)
		subnetIDs = append(subnetIDs, *subnet.SubnetId)
	}

	_, err = client.CreateTags(&ec2.CreateTagsInput{
		Resources: resourceIDs,
		Tags:      []*ec2.Tag{ec2ClusterTag(cluster.Name)},
	})
	if err != nil {
		return cluster, fmt.Errorf("failed to tag resources (one of securityGroup (%s), routeTable (%s) and/or subnets (%v)): %w",
			cluster.Spec.Cloud.AWS.SecurityGroupID, cluster.Spec.Cloud.AWS.RouteTableID, subnetIDs, err)
	}

	return update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.AddFinalizer(cluster, tagCleanupFinalizer)
	})
}

func cleanUpTags(client ec2iface.EC2API, cluster *kubermaticv1.Cluster) error {
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

	resourceIDs := []*string{}
	filters := []*ec2.Filter{
		ec2VPCFilter(cluster.Spec.Cloud.AWS.VPCID),
		{
			Name:   aws.String("tag-key"),
			Values: aws.StringSlice([]string{*tag.Key}),
		},
	}

	// list subnets (we do not create subnets, but we tagged all of them
	// to make the AWS CCM LoadBalancer integration work)
	subnets, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{Filters: filters})
	if err != nil {
		return fmt.Errorf("failed to list subnets: %w", err)
	}

	for _, subnet := range subnets.Subnets {
		resourceIDs = append(resourceIDs, subnet.SubnetId)
	}

	// list security groups
	securityGroups, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{Filters: filters})
	if err != nil {
		return fmt.Errorf("failed to list security groups: %w", err)
	}

	for _, group := range securityGroups.SecurityGroups {
		resourceIDs = append(resourceIDs, group.GroupId)
	}

	// list route tables
	routeTables, err := client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{Filters: filters})
	if err != nil {
		return fmt.Errorf("failed to list route tables: %w", err)
	}

	for _, rt := range routeTables.RouteTables {
		resourceIDs = append(resourceIDs, rt.RouteTableId)
	}

	// remove tag
	_, err = client.DeleteTags(&ec2.DeleteTagsInput{
		Resources: resourceIDs,
		Tags:      []*ec2.Tag{tag},
	})

	return err
}
