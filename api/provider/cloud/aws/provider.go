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
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/provider"
	"github.com/kubermatic/kubermatic/api/provider/template"
	"github.com/kubermatic/kubermatic/api/uuid"
)

const (
	accessKeyIDAnnotationKey     = "acccess-key-id"
	secretAccessKeyAnnotationKey = "secret-access-key"
	subnetIDKey                  = "subnet-id"
	vpcIDKey                     = "vpc-id"
	routeTableIDKey              = "route-table-id"
	roleNameKey                  = "role-name"
	instanceProfileNameKey       = "instance-profile-name"
	availabilityZoneKey          = "availability-zone"
	securityGroupKey             = "security-group"

	tplPath = "/opt/template/nodes/aws.yaml"

	policyRoute53FullAccess = "arn:aws:iam::aws:policy/AmazonRoute53FullAccess"
	policyEC2FullAccess     = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"
)

var roleARNS = []string{policyRoute53FullAccess, policyEC2FullAccess}

type amazonEc2 struct {
	dcs map[string]provider.DatacenterMeta
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

func getVpc(vpcID string, client *ec2.EC2) (*ec2.Vpc, error) {
	if vpcID != "" {
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

	return getDefaultVpc(client)
}

func getDefaultSubnet(client *ec2.EC2, vpc *ec2.Vpc, zone string) (*ec2.Subnet, error) {
	sOut, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("availability-zone"), Values: []*string{aws.String(zone)},
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
		return nil, errors.New("unable not find default subnet")
	}

	return sOut.Subnets[0], nil
}

func getSubnet(subnetID string, client *ec2.EC2, vpc *ec2.Vpc, zone string) (*ec2.Subnet, error) {
	if subnetID != "" {
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

	return getDefaultSubnet(client, vpc, zone)
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

func (a *amazonEc2) Initialize(cloud *api.CloudSpec, name string) (*api.CloudSpec, error) {
	client, err := a.getEC2client(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get EC2 client: %v", err)
	}

	vpc, err := getVpc(cloud.AWS.VPCID, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get default vpc: %v", err)
	}
	cloud.AWS.VPCID = *vpc.VpcId

	dc, ok := a.dcs[cloud.DatacenterName]
	if !ok {
		return nil, fmt.Errorf("could not find datacenter %s", cloud.DatacenterName)
	}

	subnet, err := getSubnet(cloud.AWS.SubnetID, client, vpc, dc.Spec.AWS.Region+dc.Spec.AWS.ZoneCharacter)
	if err != nil {
		return nil, fmt.Errorf("failed to get default subnet: %v", err)
	}
	cloud.AWS.SubnetID = *subnet.SubnetId
	cloud.AWS.AvailabilityZone = *subnet.AvailabilityZone

	if cloud.AWS.SecurityGroup == "" {
		securityGroup, err := addSecurityGroup(client, vpc, name)
		if err != nil {
			return nil, fmt.Errorf("failed to add security group: %v", err)
		}
		cloud.AWS.SecurityGroup = securityGroup
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

func (*amazonEc2) MarshalCloudSpec(cs *api.CloudSpec) (map[string]string, error) {
	return map[string]string{
		accessKeyIDAnnotationKey:     cs.AWS.AccessKeyID,
		secretAccessKeyAnnotationKey: cs.AWS.SecretAccessKey,
		subnetIDKey:                  cs.AWS.SubnetID,
		vpcIDKey:                     cs.AWS.VPCID,
		routeTableIDKey:              cs.AWS.RouteTableID,
		roleNameKey:                  cs.AWS.RoleName,
		instanceProfileNameKey:       cs.AWS.InstanceProfileName,
		availabilityZoneKey:          cs.AWS.AvailabilityZone,
		securityGroupKey:             cs.AWS.SecurityGroup,
	}, nil
}

func (*amazonEc2) UnmarshalCloudSpec(annotations map[string]string) (spec *api.CloudSpec, err error) {
	spec = &api.CloudSpec{
		AWS: &api.AWSCloudSpec{},
	}
	var ok bool
	if spec.AWS.AccessKeyID, ok = annotations[accessKeyIDAnnotationKey]; !ok {
		return nil, errors.New("no access key ID found")
	}

	if spec.AWS.SecretAccessKey, ok = annotations[secretAccessKeyAnnotationKey]; !ok {
		return nil, errors.New("no secret key found")
	}

	if spec.AWS.SubnetID, ok = annotations[subnetIDKey]; !ok {
		return nil, errors.New("no subnet ID found")
	}

	if spec.AWS.VPCID, ok = annotations[vpcIDKey]; !ok {
		return nil, errors.New("no vpc ID found")
	}

	if spec.AWS.RouteTableID, ok = annotations[routeTableIDKey]; !ok {
		return nil, errors.New("no route table ID found")
	}

	if spec.AWS.RoleName, ok = annotations[roleNameKey]; !ok {
		return nil, errors.New("no role ID found")
	}

	if spec.AWS.InstanceProfileName, ok = annotations[instanceProfileNameKey]; !ok {
		return nil, errors.New("no instance profile ID found")
	}

	if spec.AWS.AvailabilityZone, ok = annotations[availabilityZoneKey]; !ok {
		return nil, errors.New("no availability zone found")
	}
	spec.AWS.SecurityGroup = annotations[securityGroupKey]

	return spec, nil
}

func (a *amazonEc2) CreateNodeClass(c *api.Cluster, nSpec *api.NodeSpec, keys []extensions.UserSSHKey, version *api.MasterVersion) (*v1alpha1.NodeClass, error) {
	dc, found := a.dcs[c.Spec.Cloud.DatacenterName]
	if !found || dc.Spec.AWS == nil {
		return nil, fmt.Errorf("invalid datacenter %q", c.Spec.Cloud.DatacenterName)
	}

	nc, err := resources.LoadNodeClassFile(tplPath, a.GetNodeClassName(nSpec), c, nSpec, dc, keys, version)
	if err != nil {
		return nil, fmt.Errorf("could not load nodeclass: %v", err)
	}

	client, err := c.GetNodesetClient()
	if err != nil {
		return nil, fmt.Errorf("could not get nodeclass client: %v", err)
	}

	cnc, err := client.NodesetV1alpha1().NodeClasses().Create(nc)
	if err != nil {
		return nil, fmt.Errorf("could not create nodeclass: %v", err)
	}

	return cnc, nil
}

func (a *amazonEc2) GetNodeClassName(nSpec *api.NodeSpec) string {
	return fmt.Sprintf("kubermatic-%s", uuid.ShortUID(5))
}

func (a *amazonEc2) getSession(cloud *api.CloudSpec) (*session.Session, error) {
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

func (a *amazonEc2) getEC2client(cloud *api.CloudSpec) (*ec2.EC2, error) {
	sess, err := a.getSession(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get amazonEc2 session: %v", err)
	}
	return ec2.New(sess), nil
}

func (a *amazonEc2) getIAMClient(cloud *api.CloudSpec) (*iam.IAM, error) {
	sess, err := a.getSession(cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get amazonEc2 session: %v", err)
	}
	return iam.New(sess), nil
}

func (a *amazonEc2) CleanUp(cloud *api.CloudSpec) error {
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
