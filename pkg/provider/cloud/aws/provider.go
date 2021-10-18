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
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"
	httperror "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/klog"
)

const (
	resourceNamePrefix = "kubernetes-"

	regionAnnotationKey = "kubermatic.io/aws-region"

	securityGroupCleanupFinalizer    = "kubermatic.io/cleanup-aws-security-group"
	instanceProfileCleanupFinalizer  = "kubermatic.io/cleanup-aws-instance-profile"
	controlPlaneRoleCleanupFinalizer = "kubermatic.io/cleanup-aws-control-plane-role"
	tagCleanupFinalizer              = "kubermatic.io/cleanup-aws-tags"

	tagNameKubernetesClusterPrefix = "kubernetes.io/cluster/"

	authFailure = "AuthFailure"
)

type AmazonEC2 struct {
	dc                *kubermaticv1.DatacenterSpecAWS
	secretKeySelector provider.SecretKeySelectorValueFunc
}

type EKSClientSet struct {
	EKS eksiface.EKSAPI
	IAM iamiface.IAMAPI
}

func (a *AmazonEC2) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (a *AmazonEC2) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	client, err := a.getClientSet(spec)
	if err != nil {
		return fmt.Errorf("failed to get API client: %v", err)
	}

	// Some settings require the vpc to be set
	if spec.AWS.VPCID == "" {
		if spec.AWS.SecurityGroupID != "" {
			return fmt.Errorf("vpc must be set when specifying a security group")
		}
	}

	if spec.AWS.VPCID != "" {
		vpc, err := getVPCByID(spec.AWS.VPCID, client.EC2)
		if err != nil {
			return err
		}

		if spec.AWS.SecurityGroupID != "" {
			if _, err = getSecurityGroupByID(client.EC2, vpc, spec.AWS.SecurityGroupID); err != nil {
				return err
			}
		}
	}

	return nil
}

// MigrateToMultiAZ migrates an AWS cluster from the old AZ-hardcoded spec to multi-AZ spec
func (a *AmazonEC2) MigrateToMultiAZ(cluster *kubermaticv1.Cluster, clusterUpdater provider.ClusterUpdater) error {
	// If not even the role name is set, then the cluster is not fully
	// initialized and we don't need to worry about this migration just yet.
	if cluster.Spec.Cloud.AWS.RoleName == "" {
		return nil
	}

	if cluster.Spec.Cloud.AWS.ControlPlaneRoleARN == "" {
		client, err := a.getClientSet(cluster.Spec.Cloud)
		if err != nil {
			return fmt.Errorf("failed to get API client: %v", err)
		}

		paramsRoleGet := &iam.GetRoleInput{RoleName: aws.String(cluster.Spec.Cloud.AWS.RoleName)}
		getRoleOut, err := client.IAM.GetRole(paramsRoleGet)
		if err != nil {
			return fmt.Errorf("failed to get already existing aws IAM role %s: %v", cluster.Spec.Cloud.AWS.RoleName, err)
		}

		newCluster, err := clusterUpdater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.AWS.ControlPlaneRoleARN = *getRoleOut.Role.Arn
		})
		if err != nil {
			return err
		}
		*cluster = *newCluster
	}

	return nil
}

// AddICMPRulesIfRequired will create security rules that allow ICMP traffic if these do not yet exist.
// It is a part of a migration for older clusers (migrationRevision < 1) that didn't have these rules.
func (a *AmazonEC2) AddICMPRulesIfRequired(cluster *kubermaticv1.Cluster) error {
	if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		klog.Infof("Not adding ICMP allow rules for cluster %q as it has no securityGroupID set",
			cluster.Name)
		return nil
	}

	client, err := a.getClientSet(cluster.Spec.Cloud)
	if err != nil {
		return fmt.Errorf("failed to get API client: %v", err)
	}
	out, err := client.EC2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice([]string{cluster.Spec.Cloud.AWS.SecurityGroupID}),
	})
	if err != nil {
		return fmt.Errorf("failed to get security group %q: %v", cluster.Spec.Cloud.AWS.SecurityGroupID, err)
	}

	// Should never happen
	if len(out.SecurityGroups) > 1 {
		return fmt.Errorf("got more than one(%d) security group for id %q",
			(len(out.SecurityGroups)), cluster.Spec.Cloud.AWS.SecurityGroupID)
	}
	if len(out.SecurityGroups) == 0 {
		return fmt.Errorf("did not find a security group for id %q",
			cluster.Spec.Cloud.AWS.SecurityGroupID)
	}

	var hasIPV4ICMPRule, hasIPV6ICMPRule bool
	for _, rule := range out.SecurityGroups[0].IpPermissions {
		if rule.FromPort != nil && *rule.FromPort == -1 && rule.ToPort != nil && *rule.ToPort == -1 {

			if *rule.IpProtocol == "icmp" && len(rule.IpRanges) == 1 && *rule.IpRanges[0].CidrIp == "0.0.0.0/0" {
				hasIPV4ICMPRule = true
			}
			if *rule.IpProtocol == "icmpv6" && len(rule.Ipv6Ranges) == 1 && *rule.Ipv6Ranges[0].CidrIpv6 == "::/0" {
				hasIPV6ICMPRule = true
			}
		}
	}

	var secGroupRules []*ec2.IpPermission
	if !hasIPV4ICMPRule {
		klog.Infof("Adding allow rule for icmp to cluster %q", cluster.Name)
		secGroupRules = append(secGroupRules,
			(&ec2.IpPermission{}).
				SetIpProtocol("icmp").
				SetFromPort(-1).
				SetToPort(-1).
				SetIpRanges([]*ec2.IpRange{
					{CidrIp: aws.String("0.0.0.0/0")},
				}))
	}
	if !hasIPV6ICMPRule {
		klog.Infof("Adding allow rule for icmpv6 to cluster %q", cluster.Name)
		secGroupRules = append(secGroupRules,
			(&ec2.IpPermission{}).
				SetIpProtocol("icmpv6").
				SetFromPort(-1).
				SetToPort(-1).
				SetIpv6Ranges([]*ec2.Ipv6Range{
					{CidrIpv6: aws.String("::/0")},
				}))
	}

	if len(secGroupRules) > 0 {
		_, err = client.EC2.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       aws.String(cluster.Spec.Cloud.AWS.SecurityGroupID),
			IpPermissions: secGroupRules,
		})
		if err != nil {
			return fmt.Errorf("failed to add ICMP rules to security group %q: %v",
				cluster.Spec.Cloud.AWS.SecurityGroupID, err)
		}
	}

	return nil
}

// NewCloudProvider returns a new AmazonEC2 provider.
func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*AmazonEC2, error) {
	if dc.Spec.AWS == nil {
		return nil, errors.New("datacenter is not an AWS datacenter")
	}
	return &AmazonEC2{
		dc:                dc.Spec.AWS,
		secretKeySelector: secretKeyGetter,
	}, nil
}

func getDefaultVpc(client ec2iface.EC2API) (*ec2.Vpc, error) {
	vpcOut, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("isDefault"), Values: []*string{aws.String("true")}},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list vpc's: %v", err)
	}

	if len(vpcOut.Vpcs) != 1 {
		return nil, errors.New("unable not find default vpc")
	}

	return vpcOut.Vpcs[0], nil
}

func getRouteTable(vpcID string, client ec2iface.EC2API) (*ec2.RouteTable, error) {
	out, err := client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{&vpcID}},
			{Name: aws.String("association.main"), Values: []*string{aws.String("true")}},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(out.RouteTables) != 1 {
		return nil, fmt.Errorf("could not get default RouteTable for vpc-id: %s. Make sure you have exact one main RouteTable for the vpc", vpcID)
	}

	return out.RouteTables[0], nil
}

func getVPCByID(vpcID string, client ec2iface.EC2API) (*ec2.Vpc, error) {
	vpcOut, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(vpcID)}},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list vpc's: %v", err)
	}

	if len(vpcOut.Vpcs) != 1 {
		return nil, fmt.Errorf("unable to find specified vpc with id %q", vpcID)
	}

	return vpcOut.Vpcs[0], nil
}

func clusterTag(clusterName string) *ec2.Tag {
	return &ec2.Tag{
		Key:   aws.String(tagNameKubernetesClusterPrefix + clusterName),
		Value: aws.String(""),
	}
}

func tagResources(cluster *kubermaticv1.Cluster, client ec2iface.EC2API) error {
	sOut, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"), Values: aws.StringSlice([]string{cluster.Spec.Cloud.AWS.VPCID}),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list subnets: %v", err)
	}

	resourceIDs := []*string{&cluster.Spec.Cloud.AWS.SecurityGroupID, &cluster.Spec.Cloud.AWS.RouteTableID}
	var subnetIDs []string
	for _, subnet := range sOut.Subnets {
		resourceIDs = append(resourceIDs, subnet.SubnetId)
		subnetIDs = append(subnetIDs, *subnet.SubnetId)
	}

	_, err = client.CreateTags(&ec2.CreateTagsInput{
		Resources: resourceIDs,
		Tags:      []*ec2.Tag{clusterTag(cluster.Name)},
	})
	if err != nil {
		return fmt.Errorf("failed to tag securityGroup(id=%s), routeTable(id=%s) and subnets (ids=%v): %v",
			cluster.Spec.Cloud.AWS.SecurityGroupID, cluster.Spec.Cloud.AWS.RouteTableID, subnetIDs, err)
	}
	return nil
}

func removeTags(cluster *kubermaticv1.Cluster, client ec2iface.EC2API) error {
	sOut, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"), Values: aws.StringSlice([]string{cluster.Spec.Cloud.AWS.VPCID}),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list subnets: %v", err)
	}

	resourceIDs := []*string{&cluster.Spec.Cloud.AWS.SecurityGroupID, &cluster.Spec.Cloud.AWS.RouteTableID}
	for _, subnet := range sOut.Subnets {
		resourceIDs = append(resourceIDs, subnet.SubnetId)
	}

	_, err = client.DeleteTags(&ec2.DeleteTagsInput{
		Resources: resourceIDs,
		Tags:      []*ec2.Tag{clusterTag(cluster.Name)},
	})
	return err
}

// Get security group by aws generated id string (sg-xxxxx).
// Error is returned in case no such group exists.
func getSecurityGroupByID(client ec2iface.EC2API, vpc *ec2.Vpc, id string) (*ec2.SecurityGroup, error) {
	dsgOut, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice([]string{id}),
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{vpc.VpcId},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get security group: %v", err)
	}
	if len(dsgOut.SecurityGroups) == 0 {
		return nil, fmt.Errorf("security group with id '%s' not found in vpc %s", id, *vpc.VpcId)
	}

	return dsgOut.SecurityGroups[0], nil
}

// Create security group ("sg") with name `name` in `vpc`. The name
// in a sg must be unique within the vpc (no pre-existing sg with
// that name is allowed).
func createSecurityGroup(client ec2iface.EC2API, vpcID, clusterName string, nodeportLow, nodeportHigh int) (string, error) {
	var securityGroupID string

	newSecurityGroupName := resourceNamePrefix + clusterName
	csgOut, err := client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		VpcId:       &vpcID,
		GroupName:   aws.String(newSecurityGroupName),
		Description: aws.String(fmt.Sprintf("Security group for the Kubernetes cluster %s", clusterName)),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); !ok || awsErr.Code() != "InvalidGroup.Duplicate" {
			return "", fmt.Errorf("failed to create security group %s: %v", newSecurityGroupName, err)
		}
		describeOut, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(newSecurityGroupName)}}},
		})
		if err != nil {
			return "", fmt.Errorf("failed to get security group after creation failed because the group already existed: %v", err)
		}
		if n := len(describeOut.SecurityGroups); n != 1 {
			return "", fmt.Errorf("expected to get exactly one security group after create failed because the group already existed, got %d", n)
		}

		securityGroupID = aws.StringValue(describeOut.SecurityGroups[0].GroupId)
	}
	if csgOut != nil && csgOut.GroupId != nil {
		securityGroupID = *csgOut.GroupId
	}
	klog.V(2).Infof("Security group %s for cluster %s created with id %s.", newSecurityGroupName, clusterName, securityGroupID)

	// define permissions
	permissions := []*ec2.IpPermission{
		(&ec2.IpPermission{}).
			// all protocols from within the sg
			SetIpProtocol("-1").
			SetUserIdGroupPairs([]*ec2.UserIdGroupPair{
				(&ec2.UserIdGroupPair{}).
					SetGroupId(securityGroupID),
			}),
		(&ec2.IpPermission{}).
			// tcp:22 from everywhere
			SetIpProtocol("tcp").
			SetFromPort(provider.DefaultSSHPort).
			SetToPort(provider.DefaultSSHPort).
			SetIpRanges([]*ec2.IpRange{
				{CidrIp: aws.String("0.0.0.0/0")},
			}),
		(&ec2.IpPermission{}).
			// ICMP from/to everywhere
			SetIpProtocol("icmp").
			SetFromPort(-1). // any port
			SetToPort(-1).   // any port
			SetIpRanges([]*ec2.IpRange{
				{CidrIp: aws.String("0.0.0.0/0")},
			}),
		(&ec2.IpPermission{}).
			// ICMPv6 from/to everywhere
			SetIpProtocol("icmpv6").
			SetFromPort(-1). // any port
			SetToPort(-1).   // any port
			SetIpv6Ranges([]*ec2.Ipv6Range{
				{CidrIpv6: aws.String("::/0")},
			}),
		(&ec2.IpPermission{}).
			// tcp:nodeports in given range
			SetIpProtocol("tcp").
			SetFromPort(int64(nodeportLow)).
			SetToPort(int64(nodeportHigh)).
			SetIpRanges([]*ec2.IpRange{
				{CidrIp: aws.String("0.0.0.0/0")},
			}),
		(&ec2.IpPermission{}).
			// udp:nodeports in given range
			SetIpProtocol("udp").
			SetFromPort(int64(nodeportLow)).
			SetToPort(int64(nodeportHigh)).
			SetIpRanges([]*ec2.IpRange{
				{CidrIp: aws.String("0.0.0.0/0")},
			}),
	}

	// Iterate over the permissions and add them one by one, because if an error occurs (e.g., one permission already exists)
	// none of them will be created
	for _, perm := range permissions {
		// Add permission
		_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(securityGroupID),
			IpPermissions: []*ec2.IpPermission{
				perm,
			},
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); !ok || awsErr.Code() != "InvalidPermission.Duplicate" {
				return "", fmt.Errorf("failed to authorize security group %s with id %s: %v", newSecurityGroupName, securityGroupID, err)
			}
		}
	}

	return securityGroupID, nil
}

func (a *AmazonEC2) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	client, err := a.getClientSet(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get API client: %v", err)
	}

	if cluster.Spec.Cloud.AWS.VPCID == "" {
		vpc, err := getDefaultVpc(client.EC2)
		if err != nil {
			return nil, fmt.Errorf("failed to get default vpc: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.AWS.VPCID = *vpc.VpcId
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		lowPort, highPort := kubermaticresources.NewTemplateDataBuilder().
			WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
			WithCluster(cluster).
			Build().
			NodePorts()

		securityGroupID, err := createSecurityGroup(client.EC2, cluster.Spec.Cloud.AWS.VPCID, cluster.Name, lowPort, highPort)
		if err != nil {
			return nil, fmt.Errorf("failed to add security group for cluster %s: %v", cluster.Name, err)
		}
		if len(securityGroupID) == 0 {
			return nil, fmt.Errorf("createSecurityGroup for cluster %s did not return sg id", cluster.Name)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, securityGroupCleanupFinalizer)
			cluster.Spec.Cloud.AWS.SecurityGroupID = securityGroupID
		})
		if err != nil {
			return nil, err
		}
	}

	// We create a dedicated role for the control plane
	if cluster.Spec.Cloud.AWS.ControlPlaneRoleARN == "" {
		controlPlaneRole, err := createControlPlaneRole(client.IAM, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create control plane role: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, controlPlaneRoleCleanupFinalizer)
			cluster.Spec.Cloud.AWS.ControlPlaneRoleARN = *controlPlaneRole.RoleName
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.InstanceProfileName == "" {
		workerInstanceProfile, err := createWorkerInstanceProfile(client.IAM, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to setup the required roles/instance profiles: %v", err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, instanceProfileCleanupFinalizer)
			cluster.Spec.Cloud.AWS.InstanceProfileName = *workerInstanceProfile.InstanceProfileName
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.RouteTableID == "" {
		routeTable, err := getRouteTable(cluster.Spec.Cloud.AWS.VPCID, client.EC2)
		if err != nil {
			return nil, fmt.Errorf("failed to get default RouteTable: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.AWS.RouteTableID = *routeTable.RouteTableId
		})
		if err != nil {
			return nil, err
		}
	}

	if !kuberneteshelper.HasFinalizer(cluster, tagCleanupFinalizer) {
		if err := tagResources(cluster, client.EC2); err != nil {
			return nil, err
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, tagCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	// We put this as an annotation on the cluster to allow addons to read this
	// information.
	if cluster.Annotations[regionAnnotationKey] != a.dc.Region {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			if cluster.Annotations == nil {
				cluster.Annotations = map[string]string{}
			}
			cluster.Annotations[regionAnnotationKey] = a.dc.Region
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (accessKeyID, secretAccessKey string, err error) {
	accessKeyID = cloud.AWS.AccessKeyID
	secretAccessKey = cloud.AWS.SecretAccessKey

	if accessKeyID == "" {
		if cloud.AWS.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		accessKeyID, err = secretKeySelector(cloud.AWS.CredentialsReference, resources.AWSAccessKeyID)
		if err != nil {
			return "", "", err
		}
	}

	if secretAccessKey == "" {
		if cloud.AWS.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		secretAccessKey, err = secretKeySelector(cloud.AWS.CredentialsReference, resources.AWSSecretAccessKey)
		if err != nil {
			return "", "", err
		}
	}

	return accessKeyID, secretAccessKey, nil
}

func (a *AmazonEC2) getClientSet(cloud kubermaticv1.CloudSpec) (*ClientSet, error) {
	accessKeyID, secretAccessKey, err := GetCredentialsForCluster(cloud, a.secretKeySelector)
	if err != nil {
		return nil, err
	}

	return GetClientSet(accessKeyID, secretAccessKey, a.dc.Region)
}

func (a *AmazonEC2) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, updater provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	client, err := a.getClientSet(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get API client: %v", err)
	}

	if kuberneteshelper.HasFinalizer(cluster, securityGroupCleanupFinalizer) {
		_, err = client.EC2.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(cluster.Spec.Cloud.AWS.SecurityGroupID),
		})

		if err != nil {
			if err.(awserr.Error).Code() != "InvalidGroup.NotFound" {
				return nil, fmt.Errorf("failed to delete security group %s: %s", cluster.Spec.Cloud.AWS.SecurityGroupID, err.(awserr.Error).Message())
			}
		}
		cluster, err = updater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, securityGroupCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, instanceProfileCleanupFinalizer) {
		if err := deleteInstanceProfile(client.IAM, cluster.Spec.Cloud.AWS.InstanceProfileName); err != nil {
			return nil, fmt.Errorf("failed to delete the instance profile: %v", err)
		}

		rolesToDelete := []string{
			workerRoleName(cluster.Name),
		}
		// There was a time where we saved the role name on the cluster & deleted based on that field.
		// We now have a fixed name for the roles and delete on that.
		if cluster.Spec.Cloud.AWS.RoleName != "" {
			rolesToDelete = append(rolesToDelete, cluster.Spec.Cloud.AWS.RoleName)
		}
		for _, role := range rolesToDelete {
			if err := deleteRole(client.IAM, role); err != nil {
				return nil, fmt.Errorf("failed to delete role %q: %v", role, err)
			}
		}

		cluster, err = updater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, instanceProfileCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, controlPlaneRoleCleanupFinalizer) {
		roleName := controlPlaneRoleName(cluster.Name)
		if err := deleteRole(client.IAM, roleName); err != nil {
			return nil, fmt.Errorf("failed to delete role %q: %v", roleName, err)
		}
		cluster, err = updater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, controlPlaneRoleCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, tagCleanupFinalizer) {
		if err := removeTags(cluster, client.EC2); err != nil {
			return nil, fmt.Errorf("failed to cleanup tags: %v", err)
		}
		cluster, err = updater(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, tagCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func isEntityAlreadyExists(err error) bool {
	aerr, ok := err.(awserr.Error)
	if !ok {
		return false
	}
	return aerr.Code() == "EntityAlreadyExists"
}

func isNotFound(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "NoSuchEntity" {
			return true
		}
	}
	return false
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (a *AmazonEC2) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

// GetSubnets returns the list of subnets for a selected AWS vpc.
func GetSubnets(accessKeyID, secretAccessKey, region, vpcID string) ([]*ec2.Subnet, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	filters := []*ec2.Filter{
		{Name: aws.String("vpc-id"), Values: []*string{aws.String(vpcID)}},
	}
	subnetsInput := &ec2.DescribeSubnetsInput{Filters: filters}
	out, err := client.EC2.DescribeSubnets(subnetsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %v", err)
	}

	return out.Subnets, nil
}

// GetVPCS returns the list of AWS VPC's.
func GetVPCS(accessKeyID, secretAccessKey, region string) ([]*ec2.Vpc, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	vpcOut, err := client.EC2.DescribeVpcs(&ec2.DescribeVpcsInput{})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == authFailure {
			return nil, httperror.New(401, fmt.Sprintf("failed to list vpc's: %s", awsErr.Message()))
		}

		return nil, fmt.Errorf("failed to list vpc's: %v", err)
	}

	return vpcOut.Vpcs, nil
}

// GetSecurityGroups returns the list of AWS Security Group.
func GetSecurityGroups(accessKeyID, secretAccessKey, region string) ([]*ec2.SecurityGroup, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	sgOut, err := client.EC2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == authFailure {
			return nil, httperror.New(401, fmt.Sprintf("failed to list Security Groups: %s", awsErr.Message()))
		}

		return nil, fmt.Errorf("failed to list Security Groups: %v", err)
	}

	return sgOut.SecurityGroups, nil
}

// ConnectToEKSService establishes a service connection to the Container Engine.
func ConnectToEKSService(accessKeyID, secretAccessKey, region string) (*EKSClientSet, error) {
	config := aws.NewConfig()
	config = config.WithRegion(region)

	config = config.WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""))
	config = config.WithMaxRetries(3)

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API session: %v", err)
	}

	return &EKSClientSet{
		EKS: eks.New(sess),
	}, nil
}
