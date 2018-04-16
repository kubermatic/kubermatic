package aws

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

const (
	policyRoute53FullAccess = "arn:aws:iam::aws:policy/AmazonRoute53FullAccess"
	policyEC2FullAccess     = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"
)

var roleARNS = []string{policyRoute53FullAccess, policyEC2FullAccess}

type amazonEc2 struct {
	dcs map[string]provider.DatacenterMeta
}

func (a *amazonEc2) ValidateCloudSpec(cloud *kubermaticv1.CloudSpec) error {
	client, err := a.getEC2client(cloud)
	if err != nil {
		return err
	}

	if _, err = a.getIAMClient(cloud); err != nil {
		return err
	}

	if cloud.AWS.VPCID != "" {
		if _, err = getVPCByID(cloud.AWS.VPCID, client); err != nil {
			return err
		}
	}

	if cloud.AWS.SubnetID != "" {
		if _, err = getSubnetByID(cloud.AWS.SubnetID, client); err != nil {
			return err
		}
	}

	if cloud.AWS.VPCID == "" && cloud.AWS.SubnetID == "" {
		vpc, err := getDefaultVpc(client)
		if err != nil {
			return fmt.Errorf("failed to get default vpc: %v", err)
		}

		dc, ok := a.dcs[cloud.DatacenterName]
		if !ok {
			return fmt.Errorf("could not find datacenter %s", cloud.DatacenterName)
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

func getDefaultSubnet(client *ec2.EC2, vpc *ec2.Vpc, zone string) (*ec2.Subnet, error) {
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

	if len(sOut.Subnets) != 1 {
		return nil, errors.New("no default subnet exists")
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

func getSecurityGroup(client *ec2.EC2, vpc *ec2.Vpc, name string) (*ec2.SecurityGroup, error) {
	dsgOut, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(name)},
			},
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{vpc.VpcId},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get security group: %v", err)
	}

	return dsgOut.SecurityGroups[0], nil
}

func addSecurityGroup(client *ec2.EC2, vpc *ec2.Vpc, name string) (string, error) {
	newSecurityGroupName := fmt.Sprintf("kubermatic-%s", name)
	csgOut, err := client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		VpcId:       vpc.VpcId,
		GroupName:   aws.String(newSecurityGroupName),
		Description: aws.String(fmt.Sprintf("Security group for kubermatic cluster-%s", name)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create security group: %v", err)
	}

	// Allow node-to-node communication
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		CidrIp:     vpc.CidrBlock,
		FromPort:   aws.Int64(0),
		ToPort:     aws.Int64(65535),
		GroupId:    csgOut.GroupId,
		IpProtocol: aws.String("-1"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize security group ingress for node-to-node communication: %v", err)
	}

	// Allow SSH from everywhere
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		CidrIp:     aws.String("0.0.0.0/0"),
		FromPort:   aws.Int64(provider.DefaultSSHPort),
		ToPort:     aws.Int64(provider.DefaultSSHPort),
		GroupId:    csgOut.GroupId,
		IpProtocol: aws.String("tcp"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize security group ingress for ssh: %v", err)
	}

	// Allow kubelet 10250 from everywhere
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		CidrIp:     aws.String("0.0.0.0/0"),
		FromPort:   aws.Int64(provider.DefaultKubeletPort),
		ToPort:     aws.Int64(provider.DefaultKubeletPort),
		GroupId:    csgOut.GroupId,
		IpProtocol: aws.String("tcp"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize security group ingress for kubelet port 10250: %v", err)
	}

	// Allow UDP within the security group
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		FromPort:   aws.Int64(0),
		ToPort:     aws.Int64(65535),
		GroupId:    csgOut.GroupId,
		IpProtocol: aws.String("udp"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize security group ingress for udp: %v", err)
	}

	// Allow ICMP within the security group
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:    csgOut.GroupId,
		FromPort:   aws.Int64(-1),
		ToPort:     aws.Int64(-1),
		IpProtocol: aws.String("icmp"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize security group ingress for icmp: %v", err)
	}

	return newSecurityGroupName, nil
}

func createInstanceProfile(client *iam.IAM, name string) (*iam.Role, *iam.InstanceProfile, error) {
	kubermaticRoleName := fmt.Sprintf("kubermatic-role-%s", name)
	kubermaticInstanceProfileName := fmt.Sprintf("kubermatic-instance-profile-%s", name)

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
		RoleName: aws.String(kubermaticRoleName), // Required
	}
	rOut, err := client.CreateRole(paramsRole)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create role: %v", err)
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

	paramsInstanceProfile := &iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(kubermaticInstanceProfileName), // Required
	}
	cipOut, err := client.CreateInstanceProfile(paramsInstanceProfile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create instance profile: %v", err)
	}

	paramsAddRole := &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: aws.String(kubermaticInstanceProfileName), // Required
		RoleName:            aws.String(kubermaticRoleName),            // Required
	}
	_, err = client.AddRoleToInstanceProfile(paramsAddRole)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add role %q to instance profile %q: %v", kubermaticInstanceProfileName, kubermaticRoleName, err)
	}

	return rOut.Role, cipOut.InstanceProfile, nil
}

func isInitialized(cloud *kubermaticv1.CloudSpec) bool {
	return cloud.AWS.SubnetID != "" &&
		cloud.AWS.VPCID != "" &&
		cloud.AWS.AvailabilityZone != "" &&
		cloud.AWS.InstanceProfileName != "" &&
		cloud.AWS.RoleName != "" &&
		cloud.AWS.SecurityGroup != "" &&
		cloud.AWS.RouteTableID != "" &&
		cloud.AWS.SecurityGroupID != ""
}

func (a *amazonEc2) InitializeCloudProvider(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	if isInitialized(cloud) {
		return nil, nil
	}

	client, err := a.getEC2client(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get EC2 client: %v", err)
	}

	var vpc *ec2.Vpc
	if cloud.AWS.VPCID == "" {
		vpc, err = getDefaultVpc(client)
		if err != nil {
			return nil, fmt.Errorf("failed to get default vpc: %v", err)
		}
		cloud.AWS.VPCID = *vpc.VpcId
	} else {
		vpc, err = getVPCByID(cloud.AWS.VPCID, client)
		if err != nil {
			return nil, fmt.Errorf("failed to get vpc: %v", err)
		}
	}

	dc, ok := a.dcs[cloud.DatacenterName]
	if !ok {
		return nil, fmt.Errorf("could not find datacenter %s", cloud.DatacenterName)
	}

	if cloud.AWS.SubnetID == "" {
		subnet, err := getDefaultSubnet(client, vpc, dc.Spec.AWS.Region+dc.Spec.AWS.ZoneCharacter)
		if err != nil {
			return nil, fmt.Errorf("failed to get default subnet: %v", err)
		}
		cloud.AWS.SubnetID = *subnet.SubnetId
	}

	if cloud.AWS.AvailabilityZone == "" {
		subnet, err := getSubnetByID(cloud.AWS.SubnetID, client)
		if err != nil {
			return nil, fmt.Errorf("failed to get subnet %s: %v", cloud.AWS.SubnetID, err)
		}
		cloud.AWS.AvailabilityZone = *subnet.AvailabilityZone
	}

	if cloud.AWS.SecurityGroup == "" {
		securityGroup, err := addSecurityGroup(client, vpc, name)
		if err != nil {
			return nil, fmt.Errorf("failed to add security group: %v", err)
		}
		cloud.AWS.SecurityGroup = securityGroup
	}

	if cloud.AWS.SecurityGroupID == "" {
		securityGroup, err := getSecurityGroup(client, vpc, cloud.AWS.SecurityGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to get security group %s: %v", cloud.AWS.SecurityGroup, err)
		}
		cloud.AWS.SecurityGroupID = *securityGroup.GroupId
	}

	if cloud.AWS.RoleName == "" && cloud.AWS.InstanceProfileName == "" {
		svcIAM, err := a.getIAMClient(cloud)
		if err != nil {
			return nil, fmt.Errorf("failed to get IAM client: %v", err)
		}

		role, instanceProfile, err := createInstanceProfile(svcIAM, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create instance profile: %v", err)
		}
		cloud.AWS.RoleName = *role.RoleName
		cloud.AWS.InstanceProfileName = *instanceProfile.InstanceProfileName
	}

	if cloud.AWS.RouteTableID == "" {
		routeTable, err := getRouteTable(vpc, client)
		if err != nil {
			return nil, fmt.Errorf("failed to get default RouteTable: %v", err)
		}
		cloud.AWS.RouteTableID = *routeTable.RouteTableId
	}

	return cloud, nil
}

func (a *amazonEc2) getSession(cloud *kubermaticv1.CloudSpec) (*session.Session, error) {
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

func (a *amazonEc2) getEC2client(cloud *kubermaticv1.CloudSpec) (*ec2.EC2, error) {
	sess, err := a.getSession(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get amazonEc2 session: %v", err)
	}
	return ec2.New(sess), nil
}

func (a *amazonEc2) getIAMClient(cloud *kubermaticv1.CloudSpec) (*iam.IAM, error) {
	sess, err := a.getSession(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get amazonEc2 session: %v", err)
	}
	return iam.New(sess), nil
}

func (a *amazonEc2) CleanUpCloudProvider(cloud *kubermaticv1.CloudSpec) error {
	ec2client, err := a.getEC2client(cloud)
	if err != nil {
		return fmt.Errorf("failed to get ec2 client: %v", err)
	}

	if cloud.AWS.SecurityGroup != "" {
		_, err = ec2client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupName: aws.String(cloud.AWS.SecurityGroup),
		})

		if err != nil {
			if err.(awserr.Error).Code() != "InvalidGroup.NotFound" {
				return fmt.Errorf("failed to delete security group %s: %s", cloud.AWS.SecurityGroup, err.(awserr.Error).Message())
			}
		}
	}

	iamClient, err := a.getIAMClient(cloud)
	if err != nil {
		return fmt.Errorf("failed to get iam ec2client: %v", err)
	}

	if cloud.AWS.RoleName != "" && cloud.AWS.InstanceProfileName != "" {
		_, err := iamClient.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
			RoleName:            aws.String(cloud.AWS.RoleName),
			InstanceProfileName: aws.String(cloud.AWS.InstanceProfileName),
		})
		if err != nil {
			if err.(awserr.Error).Code() != "NoSuchEntity" {
				return fmt.Errorf("failed to remove role %s from instance profile %s: %s", cloud.AWS.RoleName, cloud.AWS.InstanceProfileName, err.(awserr.Error).Message())
			}
		}
	}

	if cloud.AWS.InstanceProfileName != "" {
		_, err := iamClient.DeleteInstanceProfile(&iam.DeleteInstanceProfileInput{InstanceProfileName: &cloud.AWS.InstanceProfileName})
		if err != nil {
			if err.(awserr.Error).Code() != "NoSuchEntity" {
				return fmt.Errorf("failed to delete InstanceProfile %s: %s", cloud.AWS.InstanceProfileName, err.(awserr.Error).Message())
			}
		}
	}

	if cloud.AWS.RoleName != "" {
		for _, arn := range roleARNS {
			paramsDetachPolicy := &iam.DetachRolePolicyInput{
				PolicyArn: aws.String(arn),
				RoleName:  aws.String(cloud.AWS.RoleName),
			}
			_, err = iamClient.DetachRolePolicy(paramsDetachPolicy)
			if err != nil {
				if err.(awserr.Error).Code() != "NoSuchEntity" {
					return fmt.Errorf("failed to detach policy %s from role %s: %s", arn, cloud.AWS.RoleName, err.(awserr.Error).Message())
				}
			}
		}

		_, err := iamClient.DeleteRole(&iam.DeleteRoleInput{RoleName: &cloud.AWS.RoleName})
		if err != nil {
			if err.(awserr.Error).Code() != "NoSuchEntity" {
				return fmt.Errorf("failed to delete Role %s: %s", cloud.AWS.RoleName, err.(awserr.Error).Message())
			}
		}
	}

	return nil
}
