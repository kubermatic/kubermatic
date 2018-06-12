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

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	policyRoute53FullAccess = "arn:aws:iam::aws:policy/AmazonRoute53FullAccess"
	policyEC2FullAccess     = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"

	securityGroupCleanupFinalizer   = "kubermatic.io/cleanup-aws-security-group"
	instanceProfileCleanupFinalizer = "kubermatic.io/cleanup-aws-instance-profile"
)

var roleARNS = []string{policyRoute53FullAccess, policyEC2FullAccess}

type amazonEc2 struct {
	dcs map[string]provider.DatacenterMeta
}

func (a *amazonEc2) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
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

	return *csgOut.GroupId, nil
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

func (a *amazonEc2) InitializeCloudProvider(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	client, err := a.getEC2client(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get EC2 client: %v", err)
	}

	if cluster.Spec.Cloud.AWS.VPCID == "" {
		vpc, err := getDefaultVpc(client)
		if err != nil {
			return nil, fmt.Errorf("failed to get default vpc: %v", err)
		}
		cluster.Spec.Cloud.AWS.VPCID = *vpc.VpcId
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
		subnet, err := getDefaultSubnet(client, vpc, dc.Spec.AWS.Region+dc.Spec.AWS.ZoneCharacter)
		if err != nil {
			return nil, fmt.Errorf("failed to get default subnet: %v", err)
		}
		cluster.Spec.Cloud.AWS.SubnetID = *subnet.SubnetId
	}

	if cluster.Spec.Cloud.AWS.AvailabilityZone == "" {
		subnet, err := getSubnetByID(cluster.Spec.Cloud.AWS.SubnetID, client)
		if err != nil {
			return nil, fmt.Errorf("failed to get subnet %s: %v", cluster.Spec.Cloud.AWS.SubnetID, err)
		}
		cluster.Spec.Cloud.AWS.AvailabilityZone = *subnet.AvailabilityZone
	}

	if cluster.Spec.Cloud.AWS.SecurityGroupID == "" {
		securityGroup, err := addSecurityGroup(client, vpc, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to add security group: %v", err)
		}
		cluster.Finalizers = append(cluster.Finalizers, securityGroupCleanupFinalizer)
		cluster.Spec.Cloud.AWS.SecurityGroupID = securityGroup
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
		cluster.Finalizers = append(cluster.Finalizers, instanceProfileCleanupFinalizer)
		cluster.Spec.Cloud.AWS.RoleName = *role.RoleName
		cluster.Spec.Cloud.AWS.InstanceProfileName = *instanceProfile.InstanceProfileName
	}

	if cluster.Spec.Cloud.AWS.RouteTableID == "" {
		routeTable, err := getRouteTable(vpc, client)
		if err != nil {
			return nil, fmt.Errorf("failed to get default RouteTable: %v", err)
		}
		cluster.Spec.Cloud.AWS.RouteTableID = *routeTable.RouteTableId
	}

	return cluster, nil
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

func (a *amazonEc2) CleanUpCloudProvider(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	finalizers := sets.NewString(cluster.Finalizers...)
	ec2client, err := a.getEC2client(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get ec2 client: %v", err)
	}

	if finalizers.Has(securityGroupCleanupFinalizer) {
		_, err = ec2client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(cluster.Spec.Cloud.AWS.SecurityGroupID),
		})

		if err != nil {
			if err.(awserr.Error).Code() != "InvalidGroup.NotFound" {
				return nil, fmt.Errorf("failed to delete security group %s: %s", cluster.Spec.Cloud.AWS.SecurityGroupID, err.(awserr.Error).Message())
			}
		}
		finalizers.Delete(securityGroupCleanupFinalizer)
		cluster.Finalizers = finalizers.List()
	}

	if finalizers.Has(instanceProfileCleanupFinalizer) {
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
				return nil, fmt.Errorf("failed to detach policy %s", *policy.PolicyName)
			}
		}

		if _, err := iamClient.DeleteRole(&iam.DeleteRoleInput{RoleName: &cluster.Spec.Cloud.AWS.RoleName}); err != nil {
			if err.(awserr.Error).Code() != iam.ErrCodeNoSuchEntityException {
				return nil, fmt.Errorf("failed to delete Role %s: %s", cluster.Spec.Cloud.AWS.RoleName, err.(awserr.Error).Message())
			}
		}
		finalizers.Delete(instanceProfileCleanupFinalizer)
		cluster.Finalizers = finalizers.List()
	}

	return cluster, nil
}
