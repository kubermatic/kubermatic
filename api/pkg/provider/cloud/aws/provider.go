package aws

import (
	"errors"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"github.com/golang/glog"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
)

const (
	resourceNamePrefix = "kubernetes-"

	policyRoute53FullAccess = "arn:aws:iam::aws:policy/AmazonRoute53FullAccess"
	policyEC2FullAccess     = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"

	securityGroupCleanupFinalizer   = "kubermatic.io/cleanup-aws-security-group"
	instanceProfileCleanupFinalizer = "kubermatic.io/cleanup-aws-instance-profile"
	tagCleanupFinalizer             = "kubermatic.io/cleanup-aws-tags"

	tagNameKubernetesClusterPrefix = "kubernetes.io/cluster/"
)

var roleARNS = []string{policyRoute53FullAccess, policyEC2FullAccess}

type AmazonEC2 struct {
	dcs map[string]*kubermaticv1.SeedDatacenter
}

func (a *AmazonEC2) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (a *AmazonEC2) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	client, err := a.getEC2client(spec)
	if err != nil {
		return err
	}

	if _, err = a.getIAMClient(spec); err != nil {
		return err
	}

	// Some settings require the vpc to be set
	if spec.AWS.VPCID == "" {
		if spec.AWS.SecurityGroupID != "" {
			return fmt.Errorf("vpc must be set when specifying a security group")
		}
		if spec.AWS.SubnetID != "" {
			return fmt.Errorf("vpc must be set when specifying a subnet")
		}
	}

	if spec.AWS.VPCID != "" {
		vpc, err := getVPCByID(spec.AWS.VPCID, client)
		if err != nil {
			return err
		}

		if spec.AWS.SubnetID != "" {
			if _, err = getSubnetByID(spec.AWS.SubnetID, client); err != nil {
				return err
			}
		}

		if spec.AWS.SecurityGroupID != "" {
			if _, err = getSecurityGroupByID(client, vpc, spec.AWS.SecurityGroupID); err != nil {
				return err
			}
		}
	}

	if spec.AWS.VPCID == "" && spec.AWS.SubnetID == "" {
		vpc, err := getDefaultVpc(client)
		if err != nil {
			return fmt.Errorf("failed to get default vpc: %v", err)
		}

		nodeDC, err := provider.NodeLocationFromSeedMap(a.dcs, spec.DatacenterName)
		if err != nil {
			return err
		}

		_, err = getDefaultSubnet(client, *vpc.VpcId, nodeDC.Spec.AWS.Region+nodeDC.Spec.AWS.ZoneCharacter)
		if err != nil {
			return fmt.Errorf("failed to get default subnet: %v", err)
		}
	}

	return nil
}

// MigrateToMultiAZ migrates an AWS cluster from the old AZ-hardcoded spec to multi-AZ spec
func (a *AmazonEC2) MigrateToMultiAZ(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) error {
	// If not even the role name is set, then the cluster is not fully
	// initialized and we don't need to worry about this migration just yet.
	if cluster.Spec.Cloud.AWS.RoleName == "" {
		return nil
	}

	if cluster.Spec.Cloud.AWS.RoleARN == "" {
		svcIAM, err := a.getIAMClient(cluster.Spec.Cloud)
		if err != nil {
			return fmt.Errorf("failed to get IAM client: %v", err)
		}

		paramsRoleGet := &iam.GetRoleInput{RoleName: aws.String(cluster.Spec.Cloud.AWS.RoleName)}
		getRoleOut, err := svcIAM.GetRole(paramsRoleGet)
		if err != nil {
			return fmt.Errorf("failed to get already existing aws IAM role %s: %v", cluster.Spec.Cloud.AWS.RoleName, err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.AWS.RoleARN = *getRoleOut.Role.Arn
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// AddICMPRulesIfRequired will create security rules that allow ICMP traffic if these do not yet exist.
// It is a part of a migration for older clusers (migrationRevision < 1) that didn't have these rules.
func (a *AmazonEC2) AddICMPRulesIfRequired(cluster *kubermaticv1.Cluster) error {
	if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		glog.Infof("Not adding ICMP allow rules for cluster %q as it has no securityGroupID set",
			cluster.Name)
		return nil
	}

	client, err := a.getEC2client(cluster.Spec.Cloud)
	if err != nil {
		return fmt.Errorf("failed to get EC2 client: %v", err)
	}
	out, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
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
		glog.Infof("Adding allow rule for icmp to cluster %q", cluster.Name)
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
		glog.Infof("Adding allow rule for icmpv6 to cluster %q", cluster.Name)
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
		_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
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
func NewCloudProvider(dcs map[string]*kubermaticv1.SeedDatacenter) *AmazonEC2 {
	return &AmazonEC2{
		dcs: dcs,
	}
}

func getDefaultVpc(client *ec2.EC2) (*ec2.Vpc, error) {
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

func getRouteTable(vpcID string, client *ec2.EC2) (*ec2.RouteTable, error) {
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

func getVPCByID(vpcID string, client *ec2.EC2) (*ec2.Vpc, error) {
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

func tagResources(cluster *kubermaticv1.Cluster, client *ec2.EC2) error {
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

	resourceIDs := []*string{&cluster.Spec.Cloud.AWS.SecurityGroupID, &cluster.Spec.Cloud.AWS.RouteTableID, &cluster.Spec.Cloud.AWS.SecurityGroupID}
	for _, subnet := range sOut.Subnets {
		resourceIDs = append(resourceIDs, subnet.SubnetId)
	}

	_, err = client.CreateTags(&ec2.CreateTagsInput{
		Resources: resourceIDs,
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(tagNameKubernetesClusterPrefix + cluster.Name),
				Value: aws.String(""),
			},
		},
	})
	return err
}

func removeTags(cluster *kubermaticv1.Cluster, client *ec2.EC2) error {
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

	resourceIDs := []*string{&cluster.Spec.Cloud.AWS.SecurityGroupID, &cluster.Spec.Cloud.AWS.RouteTableID, &cluster.Spec.Cloud.AWS.SecurityGroupID}
	for _, subnet := range sOut.Subnets {
		resourceIDs = append(resourceIDs, subnet.SubnetId)
	}

	_, err = client.DeleteTags(&ec2.DeleteTagsInput{
		Resources: resourceIDs,
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(tagNameKubernetesClusterPrefix + cluster.Name),
				Value: aws.String(""),
			},
		},
	})
	return err
}

func getDefaultSubnet(client *ec2.EC2, vpcID, zone string) (*ec2.Subnet, error) {
	glog.V(4).Infof("Looking for the default subnet for VPC %s...", vpcID)
	sOut, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("availability-zone"), Values: []*string{aws.String(zone)},
			},
			{
				Name: aws.String("defaultForAz"), Values: []*string{aws.String("true")},
			},
			{
				Name: aws.String("vpc-id"), Values: []*string{&vpcID},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %v", err)
	}

	if len(sOut.Subnets) == 0 {
		return nil, errors.New("no default subnet exists in vpc")
	}

	if len(sOut.Subnets) > 1 {
		return nil, errors.New("more than one default subnet exists in vpc")
	}

	return sOut.Subnets[0], nil
}

func getSubnetByID(subnetID string, client *ec2.EC2) (*ec2.Subnet, error) {
	sOut, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("subnet-id"), Values: []*string{aws.String(subnetID)},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %v", err)
	}

	if len(sOut.Subnets) != 1 {
		return nil, fmt.Errorf("unable to find subnet with id %q", subnetID)
	}

	return sOut.Subnets[0], nil
}

// Get security group by aws generated id string (sg-xxxxx).
// Error is returned in case no such group exists.
func getSecurityGroupByID(client *ec2.EC2, vpc *ec2.Vpc, id string) (*ec2.SecurityGroup, error) {
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
func createSecurityGroup(client *ec2.EC2, vpcID, clusterName string) (string, error) {
	newSecurityGroupName := resourceNamePrefix + clusterName
	csgOut, err := client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		VpcId:       &vpcID,
		GroupName:   aws.String(newSecurityGroupName),
		Description: aws.String(fmt.Sprintf("Security group for the Kubernetes cluster %s", clusterName)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create security group %s: %v", newSecurityGroupName, err)
	}
	sgid := aws.StringValue(csgOut.GroupId)
	glog.V(2).Infof("Security group %s for cluster %s created with id %s.", newSecurityGroupName, clusterName, sgid)

	// Add permissions.
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sgid),
		IpPermissions: []*ec2.IpPermission{
			(&ec2.IpPermission{}).
				// all protocols from within the sg
				SetIpProtocol("-1").
				SetUserIdGroupPairs([]*ec2.UserIdGroupPair{
					(&ec2.UserIdGroupPair{}).
						SetGroupId(sgid),
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
				// tcp:10250 from everywhere
				SetIpProtocol("tcp").
				SetFromPort(provider.DefaultKubeletPort).
				SetToPort(provider.DefaultKubeletPort).
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
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize security group %s with id %s: %v", newSecurityGroupName, sgid, err)
	}

	return sgid, nil
}

func createInstanceProfile(client *iam.IAM, clusterName string) (*iam.Role, *iam.InstanceProfile, error) {
	kubermaticRoleName := resourceNamePrefix + clusterName
	kubermaticInstanceProfileName := resourceNamePrefix + clusterName

	roleName := aws.String(kubermaticRoleName)
	paramsRole := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "ec2.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }
  ]
}`),
		RoleName: roleName, // Required
	}
	var role *iam.Role

	// Do the create before doing a get, because in 90% of the cases
	// this will not exist yet
	rOut, err := client.CreateRole(paramsRole)
	if err != nil {
		if !isEntityAlreadyExists(err) {
			return nil, nil, fmt.Errorf("failed to create role: %v", err)
		}
		// Accept "EntityAlreadyExists" and assume the config is correct
		paramsRoleGet := &iam.GetRoleInput{RoleName: roleName}
		getRoleOut, err := client.GetRole(paramsRoleGet)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get already existing aws IAM role %s: %v", kubermaticRoleName, err)
		}
		role = getRoleOut.Role
	} else {
		role = rOut.Role
	}

	for _, arn := range roleARNS {
		paramsAttachPolicy := &iam.AttachRolePolicyInput{
			PolicyArn: aws.String(arn),
			RoleName:  aws.String(kubermaticRoleName),
		}
		_, err = client.AttachRolePolicy(paramsAttachPolicy)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to attach role %q to policy %q: %v", kubermaticRoleName, arn, err)
		}
	}

	instanceProfileName := aws.String(kubermaticInstanceProfileName)
	paramsInstanceProfile := &iam.CreateInstanceProfileInput{
		InstanceProfileName: instanceProfileName, // Required
	}
	var instanceProfile *iam.InstanceProfile

	// Do the create before doing a get, because in 90% of the cases
	// this will not exist yet
	cipOut, err := client.CreateInstanceProfile(paramsInstanceProfile)
	if err != nil {
		if !isEntityAlreadyExists(err) {
			return nil, nil, fmt.Errorf("failed to create instance profile: %v", err)
		}
		// Accept "EntityAlreadyExists" and assume the config is correct
		paramsInstanceProfileGet := &iam.GetInstanceProfileInput{InstanceProfileName: instanceProfileName}
		getInstanceProfileOut, err := client.GetInstanceProfile(paramsInstanceProfileGet)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get already existing InstanceProfile %s: %v", *instanceProfileName, err)
		}
		instanceProfile = getInstanceProfileOut.InstanceProfile
	} else {
		instanceProfile = cipOut.InstanceProfile
	}

	// Just return if Role is already associated to InstanceProfile
	for _, role := range instanceProfile.Roles {
		if *role.RoleName == *roleName {
			return role, instanceProfile, nil
		}
	}

	paramsAddRole := &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: aws.String(kubermaticInstanceProfileName), // Required
		RoleName:            aws.String(kubermaticRoleName),            // Required
	}
	_, err = client.AddRoleToInstanceProfile(paramsAddRole)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add role %q to instance profile %q: %v", kubermaticInstanceProfileName, kubermaticRoleName, err)
	}

	return role, instanceProfile, nil
}

func (a *AmazonEC2) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	client, err := a.getEC2client(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get EC2 client: %v", err)
	}

	if cluster.Spec.Cloud.AWS.VPCID == "" {
		vpc, err := getDefaultVpc(client)
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

	nodeDC, err := provider.NodeLocationFromSeedMap(a.dcs, cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, err
	}
	if nodeDC.Spec.AWS == nil {
		return nil, fmt.Errorf("datacenter %q is not an AWS datacenter", cluster.Spec.Cloud.DatacenterName)
	}

	if cluster.Spec.Cloud.AWS.SubnetID == "" {
		glog.V(4).Infof("No Subnet specified on cluster %s", cluster.Name)
		subnet, err := getDefaultSubnet(client, cluster.Spec.Cloud.AWS.VPCID, nodeDC.Spec.AWS.Region+nodeDC.Spec.AWS.ZoneCharacter)
		if err != nil {

			return nil, fmt.Errorf("failed to get default subnet for vpc %s: %v", cluster.Spec.Cloud.AWS.VPCID, err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.AWS.SubnetID = *subnet.SubnetId
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.AvailabilityZone == "" {
		subnet, err := getSubnetByID(cluster.Spec.Cloud.AWS.SubnetID, client)
		if err != nil {
			return nil, fmt.Errorf("failed to get subnet %s: %v", cluster.Spec.Cloud.AWS.SubnetID, err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.AWS.AvailabilityZone = *subnet.AvailabilityZone
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		securityGroupID, err := createSecurityGroup(client, cluster.Spec.Cloud.AWS.VPCID, cluster.Name)
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

	if cluster.Spec.Cloud.AWS.RoleName == "" && cluster.Spec.Cloud.AWS.InstanceProfileName == "" {
		svcIAM, err := a.getIAMClient(cluster.Spec.Cloud)
		if err != nil {
			return nil, fmt.Errorf("failed to get IAM client: %v", err)
		}

		role, instanceProfile, err := createInstanceProfile(svcIAM, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create instance profile: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, instanceProfileCleanupFinalizer)
			cluster.Spec.Cloud.AWS.RoleName = *role.RoleName
			cluster.Spec.Cloud.AWS.RoleARN = *role.Arn
			cluster.Spec.Cloud.AWS.InstanceProfileName = *instanceProfile.InstanceProfileName
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.RouteTableID == "" {
		routeTable, err := getRouteTable(cluster.Spec.Cloud.AWS.VPCID, client)
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
		if err := tagResources(cluster, client); err != nil {
			return nil, err
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, tagCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func (a *AmazonEC2) getSession(cloud kubermaticv1.CloudSpec) (*session.Session, error) {
	config := aws.NewConfig()
	nodeDC, err := provider.NodeLocationFromSeedMap(a.dcs, cloud.DatacenterName)
	if err != nil {
		return nil, err
	}
	if nodeDC.Spec.AWS == nil {
		return nil, fmt.Errorf("datacenter %s is not an AWS datacenter", cloud.DatacenterName)
	}
	config = config.WithRegion(nodeDC.Spec.AWS.Region)
	config = config.WithCredentials(credentials.NewStaticCredentials(cloud.AWS.AccessKeyID, cloud.AWS.SecretAccessKey, ""))
	config = config.WithMaxRetries(3)
	return session.NewSession(config)
}

func (a *AmazonEC2) getEC2client(cloud kubermaticv1.CloudSpec) (*ec2.EC2, error) {
	sess, err := a.getSession(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get AmazonEC2 session: %v", err)
	}
	return ec2.New(sess), nil
}

func (a *AmazonEC2) getIAMClient(cloud kubermaticv1.CloudSpec) (*iam.IAM, error) {
	sess, err := a.getSession(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get AmazonEC2 session: %v", err)
	}
	return iam.New(sess), nil
}

func (a *AmazonEC2) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	ec2client, err := a.getEC2client(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get ec2 client: %v", err)
	}

	if kuberneteshelper.HasFinalizer(cluster, securityGroupCleanupFinalizer) {
		_, err = ec2client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(cluster.Spec.Cloud.AWS.SecurityGroupID),
		})

		if err != nil {
			if err.(awserr.Error).Code() != "InvalidGroup.NotFound" {
				return nil, fmt.Errorf("failed to delete security group %s: %s", cluster.Spec.Cloud.AWS.SecurityGroupID, err.(awserr.Error).Message())
			}
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, securityGroupCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, instanceProfileCleanupFinalizer) {
		iamClient, err := a.getIAMClient(cluster.Spec.Cloud)
		if err != nil {
			return nil, fmt.Errorf("failed to get iam ec2client: %v", err)
		}

		_, err = iamClient.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
			RoleName:            aws.String(cluster.Spec.Cloud.AWS.RoleName),
			InstanceProfileName: aws.String(cluster.Spec.Cloud.AWS.InstanceProfileName),
		})
		if err != nil {
			if err.(awserr.Error).Code() != iam.ErrCodeNoSuchEntityException {
				return nil, fmt.Errorf("failed to remove role %s from instance profile %s: %s", cluster.Spec.Cloud.AWS.RoleName, cluster.Spec.Cloud.AWS.InstanceProfileName, err.(awserr.Error).Message())
			}
		}

		_, err = iamClient.DeleteInstanceProfile(&iam.DeleteInstanceProfileInput{InstanceProfileName: &cluster.Spec.Cloud.AWS.InstanceProfileName})
		if err != nil {
			if err.(awserr.Error).Code() != iam.ErrCodeNoSuchEntityException {
				return nil, fmt.Errorf("failed to delete InstanceProfile %s: %s", cluster.Spec.Cloud.AWS.InstanceProfileName, err.(awserr.Error).Message())
			}
		}

		rpout, err := iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{RoleName: aws.String(cluster.Spec.Cloud.AWS.RoleName)})
		if err != nil {
			if err.(awserr.Error).Code() != iam.ErrCodeNoSuchEntityException {
				return nil, fmt.Errorf("failed to list attached role policies: %v", err)
			}
		}

		for _, policy := range rpout.AttachedPolicies {
			if _, err = iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{PolicyArn: policy.PolicyArn, RoleName: aws.String(cluster.Spec.Cloud.AWS.RoleName)}); err != nil {
				return nil, fmt.Errorf("failed to detach policy %s: %v", *policy.PolicyName, err)
			}
		}

		if _, err := iamClient.DeleteRole(&iam.DeleteRoleInput{RoleName: &cluster.Spec.Cloud.AWS.RoleName}); err != nil {
			if err.(awserr.Error).Code() != iam.ErrCodeNoSuchEntityException {
				return nil, fmt.Errorf("failed to delete Role %s: %s", cluster.Spec.Cloud.AWS.RoleName, err.(awserr.Error).Message())
			}
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, instanceProfileCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, tagCleanupFinalizer) {
		if err := removeTags(cluster, ec2client); err != nil {
			return nil, err
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
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

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (a *AmazonEC2) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}
