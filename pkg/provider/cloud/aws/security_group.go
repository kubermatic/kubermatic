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
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/utils/ptr"
)

func securityGroupName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

// Get security group by aws generated id string (sg-xxxxx).
// Error is returned in case no such group exists.
func getSecurityGroupByID(ctx context.Context, client *ec2.Client, vpc *ec2types.Vpc, id string) (*ec2types.SecurityGroup, error) {
	if vpc == nil || vpc.VpcId == nil {
		return nil, errors.New("no valid VPC given")
	}

	dsgOut, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{id},
		Filters:  []ec2types.Filter{ec2VPCFilter(ptr.Deref(vpc.VpcId, ""))},
	})
	if err != nil && !isNotFound(err) {
		return nil, fmt.Errorf("failed to get security group: %w", err)
	}

	if dsgOut == nil || len(dsgOut.SecurityGroups) == 0 {
		return nil, fmt.Errorf("security group with id '%s' not found in VPC %s", id, *vpc.VpcId)
	}

	return &dsgOut.SecurityGroups[0], nil
}

func reconcileSecurityGroup(ctx context.Context, client *ec2.Client, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	vpcID := cluster.Spec.Cloud.AWS.VPCID
	groupID := cluster.Spec.Cloud.AWS.SecurityGroupID

	// if we already have an ID on the cluster, check if that group still exists
	if groupID != "" {
		describeOut, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{groupID},
			Filters:  []ec2types.Filter{ec2VPCFilter(vpcID)},
		})
		if err != nil && !isNotFound(err) {
			return cluster, fmt.Errorf("failed to get security groups: %w", err)
		}

		// not found
		if describeOut == nil || len(describeOut.SecurityGroups) == 0 {
			groupID = ""
		}
	}

	// if no ID was stored on the cluster object or the group doesn't exist anymore,
	// try to find it by its name instead, just so we do not accidentally create
	// multiple groups with the same name (which would not be allowed by AWS)
	groupName := securityGroupName(cluster)

	if groupID == "" {
		describeOut, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			Filters: []ec2types.Filter{
				ec2VPCFilter(vpcID),
				{
					Name:   ptr.To("group-name"),
					Values: []string{groupName},
				},
			},
		})
		if err != nil {
			return cluster, fmt.Errorf("failed to get security groups: %w", err)
		}

		// found the group by its name!
		if len(describeOut.SecurityGroups) >= 1 {
			groupID = ptr.Deref(describeOut.SecurityGroups[0].GroupId, "")
		}
	}

	// if we still have no ID, we must create a new group
	if groupID == "" {
		out, err := client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			VpcId:       &vpcID,
			GroupName:   ptr.To(groupName),
			Description: ptr.To(fmt.Sprintf("Security group for the Kubernetes cluster %s", cluster.Name)),
			TagSpecifications: []ec2types.TagSpecification{{
				ResourceType: ec2types.ResourceTypeSecurityGroup,
				Tags: []ec2types.Tag{
					ec2OwnershipTag(cluster.Name),
				},
			}},
		})
		if err != nil {
			return cluster, fmt.Errorf("failed to create security group %s: %w", groupName, err)
		}

		groupID = *out.GroupId
	}

	ipv4Permissions := cluster.IsIPv4Only() || cluster.IsDualStack()
	ipv6Permissions := cluster.IsIPv6Only() || cluster.IsDualStack()

	permissions := getCommonSecurityGroupPermissions(groupID, ipv4Permissions, ipv6Permissions)

	lowPort, highPort := getNodePortRange(cluster)
	nodePortsAllowedIPRanges := kubermaticresources.GetNodePortsAllowedIPRanges(cluster, cluster.Spec.Cloud.AWS.NodePortsAllowedIPRanges, cluster.Spec.Cloud.AWS.NodePortsAllowedIPRange, nil)

	permissions = append(permissions, getNodePortSecurityGroupPermissions(lowPort, highPort, nodePortsAllowedIPRanges.GetIPv4CIDRs(), nodePortsAllowedIPRanges.GetIPv6CIDRs())...)

	// Iterate over the permissions and add them one by one, because if an error occurs
	// (e.g., one permission already exists) none of them would be created
	for _, perm := range permissions {
		// try to add permission
		_, err := client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: ptr.To(groupID),
			IpPermissions: []ec2types.IpPermission{
				perm,
			},
		})
		if err != nil {
			var awsErr smithy.APIError
			if !errors.As(err, &awsErr) || awsErr.ErrorCode() != "InvalidPermission.Duplicate" {
				return cluster, fmt.Errorf("failed to authorize security group %s with id %s: %w", groupName, groupID, err)
			}
		}
	}

	return update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Cloud.AWS.SecurityGroupID = groupID
	})
}

func getNodePortRange(cluster *kubermaticv1.Cluster) (int, int) {
	return kubermaticresources.NewTemplateDataBuilder().
		WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
		WithCluster(cluster).
		Build().
		NodePorts()
}

func getCommonSecurityGroupPermissions(securityGroupID string, ipv4Permissions, ipv6Permissions bool) []ec2types.IpPermission {
	permissions := []ec2types.IpPermission{
		// all protocols from within the sg
		{
			IpProtocol: ptr.To("-1"),
			UserIdGroupPairs: []ec2types.UserIdGroupPair{{
				GroupId: ptr.To(securityGroupID),
			}},
		},
	}

	// tcp:22 from everywhere
	sshPermission := ec2types.IpPermission{
		IpProtocol: ptr.To("tcp"),
		FromPort:   ptr.To[int32](provider.DefaultSSHPort),
		ToPort:     ptr.To[int32](provider.DefaultSSHPort),
	}
	if ipv4Permissions {
		sshPermission.IpRanges = []ec2types.IpRange{{
			CidrIp: ptr.To(kubermaticresources.IPv4MatchAnyCIDR),
		}}
	}
	if ipv6Permissions {
		sshPermission.Ipv6Ranges = []ec2types.Ipv6Range{{
			CidrIpv6: ptr.To(kubermaticresources.IPv6MatchAnyCIDR),
		}}
	}
	permissions = append(permissions, sshPermission)

	// ICMP (v4) from/to everywhere
	if ipv4Permissions {
		permissions = append(permissions, ec2types.IpPermission{
			IpProtocol: ptr.To("icmp"),
			FromPort:   ptr.To[int32](-1), // any port
			ToPort:     ptr.To[int32](-1), // any port
			IpRanges: []ec2types.IpRange{{
				CidrIp: ptr.To(kubermaticresources.IPv4MatchAnyCIDR),
			}},
		})
	}

	// ICMPv6 from/to everywhere
	if ipv6Permissions {
		permissions = append(permissions, ec2types.IpPermission{
			IpProtocol: ptr.To("icmpv6"),
			FromPort:   ptr.To[int32](-1), // any port
			ToPort:     ptr.To[int32](-1), // any port
			Ipv6Ranges: []ec2types.Ipv6Range{{
				CidrIpv6: ptr.To(kubermaticresources.IPv6MatchAnyCIDR),
			}},
		})
	}

	return permissions
}

func getNodePortSecurityGroupPermissions(lowPort, highPort int, ipv4IPRanges, ipv6IPRanges []string) []ec2types.IpPermission {
	tcpNodePortPermission := ec2types.IpPermission{
		IpProtocol: ptr.To("tcp"),
		FromPort:   ptr.To[int32](int32(lowPort)),
		ToPort:     ptr.To[int32](int32(highPort)),
	}

	udpNodePortPermission := ec2types.IpPermission{
		IpProtocol: ptr.To("udp"),
		FromPort:   ptr.To[int32](int32(lowPort)),
		ToPort:     ptr.To[int32](int32(highPort)),
	}

	for _, cidr := range ipv4IPRanges {
		tcpNodePortPermission.IpRanges = append(tcpNodePortPermission.IpRanges, ec2types.IpRange{
			CidrIp: ptr.To(cidr),
		})
		udpNodePortPermission.IpRanges = append(udpNodePortPermission.IpRanges, ec2types.IpRange{
			CidrIp: ptr.To(cidr),
		})
	}
	for _, cidr := range ipv6IPRanges {
		tcpNodePortPermission.Ipv6Ranges = append(tcpNodePortPermission.Ipv6Ranges, ec2types.Ipv6Range{
			CidrIpv6: ptr.To(cidr),
		})
		udpNodePortPermission.Ipv6Ranges = append(udpNodePortPermission.Ipv6Ranges, ec2types.Ipv6Range{
			CidrIpv6: ptr.To(cidr),
		})
	}

	return []ec2types.IpPermission{tcpNodePortPermission, udpNodePortPermission}
}

func cleanUpSecurityGroup(ctx context.Context, client *ec2.Client, cluster *kubermaticv1.Cluster) error {
	vpcID := cluster.Spec.Cloud.AWS.VPCID
	groupID := cluster.Spec.Cloud.AWS.SecurityGroupID

	// if we don't have the ID on the cluster object, try our best to find any
	// orphaned security groups by their names and still clean up as best as we can
	if groupID == "" {
		groupName := securityGroupName(cluster)

		describeOut, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			Filters: []ec2types.Filter{
				ec2VPCFilter(vpcID),
				{
					Name:   ptr.To("group-name"),
					Values: []string{groupName},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to get security groups: %w", err)
		}

		// We found a group with a matching name!
		if len(describeOut.SecurityGroups) > 0 {
			groupID = *describeOut.SecurityGroups[0].GroupId
		}
	}

	// if we still have no group ID, there is nothing to do for us
	if groupID == "" {
		return nil
	}

	// check if we own the security group
	describeOut, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{groupID},
		Filters:  []ec2types.Filter{ec2VPCFilter(vpcID)},
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get security groups: %w", err)
	}

	// someone has already deleted the group
	if describeOut == nil || len(describeOut.SecurityGroups) == 0 {
		return nil
	}

	// check if we own the SG
	if !hasEC2Tag(ec2OwnershipTag(cluster.Name), describeOut.SecurityGroups[0].Tags) {
		return nil
	}

	// time to delete the group
	_, err = client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: &groupID})

	return err
}
