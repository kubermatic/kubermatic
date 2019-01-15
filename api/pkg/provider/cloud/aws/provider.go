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
	policyRoute53FullAccess = "arn:aws:iam::aws:policy/AmazonRoute53FullAccess"
	policyEC2FullAccess     = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"

	securityGroupCleanupFinalizer   = "kubermatic.io/cleanup-aws-security-group"
	instanceProfileCleanupFinalizer = "kubermatic.io/cleanup-aws-instance-profile"
	tagCleanupFinalizer             = "kubermatic.io/cleanup-aws-tags"

	tagNameKubernetesClusterPrefix = "kubernetes.io/cluster/"
)

var roleARNS = []string{policyRoute53FullAccess, policyEC2FullAccess}

type amazonEc2 struct {
	dcs map[string]provider.DatacenterMeta
}

func (a *amazonEc2) DefaultCloudSpec(spec kubermaticv1.CloudSpec) error {
	return nil
}

func (a *amazonEc2) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
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

		dc, ok := a.dcs[spec.DatacenterName]
		if !ok {
			return fmt.Errorf("could not find datacenter %s", spec.DatacenterName)
		}

		_, err = getDefaultSubnet(client, vpc, dc.Spec.AWS.Region+dc.Spec.AWS.ZoneCharacter)
		if err != nil {
			return fmt.Errorf("failed to get default subnet: %v", err)
		}
	}

	return nil
}

// NewCloudProvider returns a new amazonEc2 provider.
func NewCloudProvider(datacenters map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &amazonEc2{
		dcs: datacenters,
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

func getRouteTable(vpc *ec2.Vpc, client *ec2.EC2) (*ec2.RouteTable, error) {
	out, err := client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{vpc.VpcId}},
			{Name: aws.String("association.main"), Values: []*string{aws.String("true")}},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(out.RouteTables) != 1 {
		return nil, errors.New("could not get default RouteTable for vpc-id:%s. Make sure you have exact one main RouteTable for the vpc")
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

func getDefaultSubnet(client *ec2.EC2, vpc *ec2.Vpc, zone string) (*ec2.Subnet, error) {
	glog.V(4).Infof("Looking for the default subnet for VPC %s...", *vpc.VpcId)
	sOut, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("availability-zone"), Values: []*string{aws.String(zone)},
			},
			{
				Name: aws.String("defaultForAz"), Values: []*string{aws.String("true")},
			},
			{
				Name: aws.String("vpc-id"), Values: []*string{vpc.VpcId},
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
func createSecurityGroup(client *ec2.EC2, vpc *ec2.Vpc, name string) (string, error) {
	newSecurityGroupName := fmt.Sprintf("kubermatic-%s", name)
	csgOut, err := client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		VpcId:       vpc.VpcId,
		GroupName:   aws.String(newSecurityGroupName),
		Description: aws.String(fmt.Sprintf("Security group for kubermatic cluster-%s", name)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create security group %s: %v", newSecurityGroupName, err)
	}
	sgid := aws.StringValue(csgOut.GroupId)
	glog.V(6).Infof("Security group %s created with id %s.", newSecurityGroupName, sgid)

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
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize security group %s with id %s: %v", newSecurityGroupName, sgid, err)
	}

	return sgid, nil
}

func createInstanceProfile(client *iam.IAM, name string) (*iam.Role, *iam.InstanceProfile, error) {
	kubermaticRoleName := fmt.Sprintf("kubermatic-role-%s", name)
	kubermaticInstanceProfileName := fmt.Sprintf("kubermatic-instance-profile-%s", name)

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

func (a *amazonEc2) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
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

	vpc, err := getVPCByID(cluster.Spec.Cloud.AWS.VPCID, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get vpc: %v", err)
	}

	dc, ok := a.dcs[cluster.Spec.Cloud.DatacenterName]
	if !ok {
		return nil, fmt.Errorf("could not find datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	if cluster.Spec.Cloud.AWS.SubnetID == "" {
		glog.V(4).Infof("No Subnet specified on cluster %s", cluster.Name)
		subnet, err := getDefaultSubnet(client, vpc, dc.Spec.AWS.Region+dc.Spec.AWS.ZoneCharacter)
		if err != nil {

			return nil, fmt.Errorf("failed to get default subnet for vpc %s: %v", *vpc.VpcId, err)
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
		securityGroupID, err := createSecurityGroup(client, vpc, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to add security group for cluster %s: %v", cluster.Name, err)
		}
		if len(securityGroupID) == 0 {
			return nil, fmt.Errorf("createSecurityGroup for cluster %s did not return sg id", cluster.Name)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, securityGroupCleanupFinalizer)
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
			cluster.Finalizers = append(cluster.Finalizers, instanceProfileCleanupFinalizer)
			cluster.Spec.Cloud.AWS.RoleName = *role.RoleName
			cluster.Spec.Cloud.AWS.InstanceProfileName = *instanceProfile.InstanceProfileName
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.AWS.RouteTableID == "" {
		routeTable, err := getRouteTable(vpc, client)
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

	if err := tagResources(cluster, client); err != nil {
		return nil, err
	}
	if !kuberneteshelper.HasFinalizer(cluster, tagCleanupFinalizer) {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, tagCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func (a *amazonEc2) getSession(cloud kubermaticv1.CloudSpec) (*session.Session, error) {
	config := aws.NewConfig()
	dc, found := a.dcs[cloud.DatacenterName]
	if !found || dc.Spec.AWS == nil {
		return nil, fmt.Errorf("can't find datacenter %s", cloud.DatacenterName)
	}
	config = config.WithRegion(dc.Spec.AWS.Region)
	config = config.WithCredentials(credentials.NewStaticCredentials(cloud.AWS.AccessKeyID, cloud.AWS.SecretAccessKey, ""))
	config = config.WithMaxRetries(3)
	return session.NewSession(config)
}

func (a *amazonEc2) getEC2client(cloud kubermaticv1.CloudSpec) (*ec2.EC2, error) {
	sess, err := a.getSession(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get amazonEc2 session: %v", err)
	}
	return ec2.New(sess), nil
}

func (a *amazonEc2) getIAMClient(cloud kubermaticv1.CloudSpec) (*iam.IAM, error) {
	sess, err := a.getSession(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get amazonEc2 session: %v", err)
	}
	return iam.New(sess), nil
}

func (a *amazonEc2) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
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
			cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, securityGroupCleanupFinalizer)
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
			cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, instanceProfileCleanupFinalizer)
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
			cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, tagCleanupFinalizer)
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
