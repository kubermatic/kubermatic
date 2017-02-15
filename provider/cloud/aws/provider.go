package aws

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"text/template"
	"time"

	sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	ktemplate "github.com/kubermatic/api/template"
	"golang.org/x/net/context"
)

const (
	// VPCCidrBlock is the default CIDR block in the VpcSubnet
	VPCCidrBlock = "10.10.0.0/16"
	// SubnetCidrBlock is the default CIDR for the VPC
	SubnetCidrBlock = "10.10.10.0/24"
)

const (
	accessKeyIDAnnotationKey     = "acccess-key-id"
	secretAccessKeyAnnotationKey = "secret-access-key"
	sshKeyNameKey                = "ssh-key-fingerprint"
	subnetIDKey                  = "subnet-id"
	vpcIDKey                     = "vpc-id"
	internetGatewayIDKey         = "internet-gateway-id"
	routeTableIDKey              = "route-table-id"
	roleNameKey                  = "role-name"
	instanceProfileNameKey       = "instance-profile-name"
	policyNameKey                = "policy-name"
	availabilityZoneKey          = "availability-zone"
	securityGroupIDKey           = "custom-security-group-id-Key"
)

const (
	awsFilterName    = "Name"
	awsFilterState   = "instance-state-name"
	awsFilterRunning = "running"
	awsFilterPending = "pending"
)

const (
	tplPath = "template/coreos/aws-cloud-config-node.yaml"
)

var (
	defaultKubermaticClusterNameTagKey = "kubermatic-cluster-name"
	defaultKubermaticClusterIDTagKey   = "kubermatic-cluster-id"
)

type aws struct {
	datacenters map[string]provider.DatacenterMeta
}

// NewCloudProvider returns a new aws provider.
func NewCloudProvider(datacenters map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &aws{
		datacenters: datacenters,
	}
}

func getDefaultVpc(client *ec2.EC2) (*ec2.Vpc, error) {
	vpcOut, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{Name: sdk.String("isDefault"), Values: []*string{sdk.String("true")}},
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

func createVpc(client *ec2.EC2) (*ec2.Vpc, error) {
	vReq := &ec2.CreateVpcInput{
		CidrBlock:       sdk.String(VPCCidrBlock),
		InstanceTenancy: sdk.String(ec2.TenancyDefault),
	}
	vpcOut, err := client.CreateVpc(vReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create vpc on ec2: %v", err)
	}

	return vpcOut.Vpc, nil
}

func getDefaultSubnet(client *ec2.EC2, vpc *ec2.Vpc, zone string) (*ec2.Subnet, error) {
	sOut, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: sdk.String("availability-zone"), Values: []*string{sdk.String(zone)},
			},
			{
				Name: sdk.String("vpc-id"), Values: []*string{vpc.VpcId},
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

func createSubnet(client *ec2.EC2, vpc *ec2.Vpc) (*ec2.Subnet, error) {
	sOut, err := client.CreateSubnet(&ec2.CreateSubnetInput{
		CidrBlock: sdk.String(SubnetCidrBlock),
		VpcId:     vpc.VpcId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet on ec2: %v", err)
	}

	return sOut.Subnet, nil
}

func createInternetGateway(client *ec2.EC2, vpc *ec2.Vpc) (*ec2.InternetGateway, error) {
	igOut, err := client.CreateInternetGateway(&ec2.CreateInternetGatewayInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to create internet gateway on ec2: %v", err)
	}

	_, err = client.AttachInternetGateway(&ec2.AttachInternetGatewayInput{
		InternetGatewayId: igOut.InternetGateway.InternetGatewayId,
		VpcId:             vpc.VpcId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach internetgateway %q to vpc %q: %v", *igOut.InternetGateway.InternetGatewayId, *vpc.VpcId, err)
	}

	return igOut.InternetGateway, nil
}

func addRoute(client *ec2.EC2, vpc *ec2.Vpc, gateway *ec2.InternetGateway) (*ec2.RouteTable, error) {
	rtOut, err := client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: sdk.String("vpc-id"), Values: []*string{vpc.VpcId}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list route tables: %v", err)
	}

	if len(rtOut.RouteTables) != 1 {
		return nil, errors.New("Could not find main RouteTable")
	}

	_, err = client.CreateRoute(&ec2.CreateRouteInput{
		GatewayId:            gateway.InternetGatewayId,
		DestinationCidrBlock: sdk.String("0.0.0.0/0"),
		RouteTableId:         rtOut.RouteTables[0].RouteTableId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create route: %v", err)
	}

	return rtOut.RouteTables[0], nil
}

func addSecurityGroup(client *ec2.EC2, vpc *ec2.Vpc, name string) (*string, error) {
	newSecurityGroupName := fmt.Sprintf("kubermatic-%s", name)
	csgOut, err := client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		VpcId:       vpc.VpcId,
		GroupName:   sdk.String(newSecurityGroupName),
		Description: sdk.String(fmt.Sprintf("Security group for kubermatic cluster-%s", name)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %v", err)
	}

	// Allow SSH from everywhere
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		CidrIp:     sdk.String("0.0.0.0/0"),
		FromPort:   sdk.Int64(22),
		ToPort:     sdk.Int64(22),
		GroupId:    csgOut.GroupId,
		IpProtocol: sdk.String("tcp"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to authorize security group ingress for ssh: %v", err)
	}

	// Allow UDP within the security group
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		FromPort:   sdk.Int64(0),
		ToPort:     sdk.Int64(65535),
		GroupId:    csgOut.GroupId,
		IpProtocol: sdk.String("udp"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to authorize security group ingress for udp: %v", err)
	}

	// Allow ICMP within the security group
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:    csgOut.GroupId,
		FromPort:   sdk.Int64(-1),
		ToPort:     sdk.Int64(-1),
		IpProtocol: sdk.String("icmp"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to authorize security group ingress for icmp: %v", err)
	}

	return csgOut.GroupId, nil
}

func getACL(client *ec2.EC2, vpc *ec2.Vpc) (*ec2.NetworkAcl, error) {
	aOut, err := client.DescribeNetworkAcls(&ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{
			{Name: sdk.String("vpc-id"), Values: []*string{vpc.VpcId}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list Acls: %v", err)
	}

	if len(aOut.NetworkAcls) != 1 {
		return nil, errors.New("Could not find main NetworkACL")
	}

	return aOut.NetworkAcls[0], nil
}

func createTags(client *ec2.EC2, cluster *api.Cluster, resources []*string) error {
	_, err := client.CreateTags(&ec2.CreateTagsInput{
		Resources: resources,
		Tags: []*ec2.Tag{
			{
				Key:   sdk.String(defaultKubermaticClusterIDTagKey),
				Value: sdk.String(cluster.Metadata.UID),
			},
			{
				Key:   sdk.String(defaultKubermaticClusterNameTagKey),
				Value: sdk.String(cluster.Metadata.Name),
			},
			{
				Key:   sdk.String(awsFilterName),
				Value: sdk.String(fmt.Sprintf("kubermatic-%s", cluster.Metadata.Name)),
			},
		},
	})

	return fmt.Errorf("failed to create tags on ec2: %v", err)
}

func createInstanceProfile(client *iam.IAM, cluster *api.Cluster) (*iam.Role, *iam.Policy, *iam.InstanceProfile, error) {
	kubermaticPolicyName := fmt.Sprintf("kubermatic-policy-%s", cluster.Metadata.Name)
	kubermaticRoleName := fmt.Sprintf("kubermatic-role-%s", cluster.Metadata.Name)
	kubermaticInstanceProfileName := fmt.Sprintf("kubermatic-instance-profile-%s", cluster.Metadata.Name)
	paramsPolicy := &iam.CreatePolicyInput{
		PolicyDocument: sdk.String(`{
   "Version": "2012-10-17",
    "Statement": [
	{
	    "Effect": "Allow",
	    "Action": "s3:*",
	    "Resource": "arn:aws:s3:::kubernetes-*"
	},
	{
	    "Effect": "Allow",
	    "Action": [
		"ec2:*",
		"route53:*",
		"ecr:GetAuthorizationToken",
		"ecr:BatchCheckLayerAvailability",
		"ecr:GetDownloadUrlForLayer",
		"ecr:GetRepositoryPolicy",
		"ecr:DescribeRepositories",
		"ecr:ListImages",
		"ecr:BatchGetImage",
		"elasticloadbalancing:*"
	    ],
	    "Resource": "*"
	}
    ]
}`), // Required
		PolicyName: sdk.String(kubermaticPolicyName), // Required
	}
	policyResp, err := client.CreatePolicy(paramsPolicy)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create policy: %v", err)
	}

	policyArn := *policyResp.Policy.Arn

	paramsRole := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: sdk.String(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "ec2.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }
  ]
}`), // Required
		RoleName: sdk.String(kubermaticRoleName), // Required
	}
	rOut, err := client.CreateRole(paramsRole)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create role: %v", err)
	}

	// Attach policy to role
	paramsAttachPolicy := &iam.AttachRolePolicyInput{
		PolicyArn: sdk.String(policyArn),          // Required
		RoleName:  sdk.String(kubermaticRoleName), // Required
	}
	_, err = client.AttachRolePolicy(paramsAttachPolicy)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to attach role %q to policy %q: %v", kubermaticRoleName, policyArn, err)
	}

	paramsInstanceProfile := &iam.CreateInstanceProfileInput{
		InstanceProfileName: sdk.String(kubermaticInstanceProfileName), // Required
	}
	cipOut, err := client.CreateInstanceProfile(paramsInstanceProfile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create instance profile: %v", err)
	}

	paramsAddRole := &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: sdk.String(kubermaticInstanceProfileName), // Required
		RoleName:            sdk.String(kubermaticRoleName),            // Required
	}
	_, err = client.AddRoleToInstanceProfile(paramsAddRole)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to add role %q to instance profile %q: %v", kubermaticInstanceProfileName, kubermaticRoleName, err)
	}

	return rOut.Role, policyResp.Policy, cipOut.InstanceProfile, nil
}

func (a *aws) InitializeCloudSpecWithDefault(cluster *api.Cluster) error {
	if cluster.Spec.Cloud.AWS.VPCId != "" {
		return nil
	}

	client, err := a.getEC2client(cluster)
	if err != nil {
		return fmt.Errorf("failed to get EC2 client: %v", err)
	}

	vpc, err := getDefaultVpc(client)
	if err != nil {
		return fmt.Errorf("failed to get default vpc: %v", err)
	}
	cluster.Spec.Cloud.AWS.VPCId = *vpc.VpcId

	dc, ok := a.datacenters[cluster.Spec.Cloud.DatacenterName]
	if !ok {
		return fmt.Errorf("could not find datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	subnet, err := getDefaultSubnet(client, vpc, dc.Spec.AWS.Zone)
	if err != nil {
		return fmt.Errorf("failed to get default subnet: %v", err)
	}
	cluster.Spec.Cloud.AWS.SubnetID = *subnet.SubnetId
	cluster.Spec.Cloud.AWS.AvailabilityZone = *subnet.AvailabilityZone

	securityGroupID, err := addSecurityGroup(client, vpc, cluster.Metadata.Name)
	if err != nil {
		return fmt.Errorf("failed to add security group: %v", err)
	}
	cluster.Spec.Cloud.AWS.SecurityGroupID = *securityGroupID

	svcIAM, err := a.getIAMclient(cluster)
	if err != nil {
		return fmt.Errorf("failed to get IAM client: %v", err)
	}

	role, policy, instanceProfile, err := createInstanceProfile(svcIAM, cluster)
	if err != nil {
		return fmt.Errorf("failed to create instance profile: %v", err)
	}
	cluster.Spec.Cloud.AWS.PolicyName = *policy.Arn
	cluster.Spec.Cloud.AWS.RoleName = *role.RoleName
	cluster.Spec.Cloud.AWS.InstanceProfileName = *instanceProfile.InstanceProfileName

	return nil
}

func (a *aws) InitializeCloudSpecWithCreate(cluster *api.Cluster) error {
	if cluster.Spec.Cloud.AWS.VPCId != "" {
		return nil
	}

	client, err := a.getEC2client(cluster)
	if err != nil {
		return fmt.Errorf("failed to get EC2 client: %v", err)
	}

	vpc, err := createVpc(client)
	if err != nil {
		return fmt.Errorf("failed to create vpc: %v", err)
	}
	cluster.Spec.Cloud.AWS.VPCId = *vpc.VpcId

	subnet, err := createSubnet(client, vpc)
	if err != nil {
		return fmt.Errorf("failed to create subnet: %v", err)
	}
	cluster.Spec.Cloud.AWS.SubnetID = *subnet.SubnetId
	cluster.Spec.Cloud.AWS.AvailabilityZone = *subnet.AvailabilityZone

	gateway, err := createInternetGateway(client, vpc)
	if err != nil {
		return fmt.Errorf("failed to create internet gateway: %v", err)
	}
	cluster.Spec.Cloud.AWS.InternetGatewayID = *gateway.InternetGatewayId

	routeTable, err := addRoute(client, vpc, gateway)
	if err != nil {
		return fmt.Errorf("failed to add route: %v", err)
	}
	cluster.Spec.Cloud.AWS.RouteTableID = *routeTable.RouteTableId

	securityGroupID, err := addSecurityGroup(client, vpc, cluster.Metadata.Name)
	if err != nil {
		return fmt.Errorf("failed to add security group: %v", err)
	}
	cluster.Spec.Cloud.AWS.SecurityGroupID = *securityGroupID

	acl, err := getACL(client, vpc)
	if err != nil {
		return fmt.Errorf("failed to get acl: %v", err)
	}

	err = createTags(client, cluster, []*string{vpc.VpcId, gateway.InternetGatewayId, subnet.SubnetId, routeTable.RouteTableId, securityGroupID, acl.NetworkAclId})
	if err != nil {
		return fmt.Errorf("failed to create tags: %v", err)
	}

	svcIAM, err := a.getIAMclient(cluster)
	if err != nil {
		return fmt.Errorf("failed to get iam client: %v", err)
	}

	role, policy, instanceProfile, err := createInstanceProfile(svcIAM, cluster)
	if err != nil {
		return fmt.Errorf("failed to create instance profile: %v", err)
	}
	cluster.Spec.Cloud.AWS.PolicyName = *policy.Arn
	cluster.Spec.Cloud.AWS.RoleName = *role.RoleName
	cluster.Spec.Cloud.AWS.InstanceProfileName = *instanceProfile.InstanceProfileName

	return nil
}

func (a *aws) InitializeCloudSpec(cluster *api.Cluster) error {
	glog.Infof("using init cloud spec mode: %s (default=use-defaults)", cluster.Spec.Cloud.AWS.InitMode)
	switch cluster.Spec.Cloud.AWS.InitMode {
	case api.AWSInitUseDefaults:
		err := a.InitializeCloudSpecWithDefault(cluster)
		if err != nil {
			return fmt.Errorf("failed to initialize cloud provider with default components: %v", err)
		}
		return nil
	case api.AWSInitCreateVpc:
		err := a.InitializeCloudSpecWithCreate(cluster)
		if err != nil {
			return fmt.Errorf("failed to initialize cloud provider with creating components: %v", err)
		}
		return nil
	default:
		err := a.InitializeCloudSpecWithDefault(cluster)
		if err != nil {
			return fmt.Errorf("failed to initialize cloud provider with default components: %v", err)
		}
		return nil
	}
}

func (*aws) MarshalCloudSpec(cs *api.CloudSpec) (map[string]string, error) {
	return map[string]string{
		accessKeyIDAnnotationKey:     cs.AWS.AccessKeyID,
		secretAccessKeyAnnotationKey: cs.AWS.SecretAccessKey,
		sshKeyNameKey:                cs.AWS.SSHKeyName,
		subnetIDKey:                  cs.AWS.SubnetID,
		vpcIDKey:                     cs.AWS.VPCId,
		internetGatewayIDKey:         cs.AWS.InternetGatewayID,
		routeTableIDKey:              cs.AWS.RouteTableID,
		roleNameKey:                  cs.AWS.RoleName,
		instanceProfileNameKey:       cs.AWS.InstanceProfileName,
		policyNameKey:                cs.AWS.PolicyName,
		availabilityZoneKey:          cs.AWS.AvailabilityZone,
		securityGroupIDKey:           cs.AWS.SecurityGroupID,
	}, nil
}

func (*aws) UnmarshalCloudSpec(annotations map[string]string) (*api.CloudSpec, error) {
	spec := &api.CloudSpec{
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

	if spec.AWS.VPCId, ok = annotations[vpcIDKey]; !ok {
		return nil, errors.New("no vpc ID found")
	}

	if spec.AWS.InternetGatewayID, ok = annotations[internetGatewayIDKey]; !ok {
		return nil, errors.New("no internet gateway ID found")
	}

	if spec.AWS.RouteTableID, ok = annotations[routeTableIDKey]; !ok {
		return nil, errors.New("no route table ID found")
	}

	if spec.AWS.SSHKeyName, ok = annotations[sshKeyNameKey]; !ok {
		return nil, errors.New("no ssh key name found")
	}

	if spec.AWS.RoleName, ok = annotations[roleNameKey]; !ok {
		return nil, errors.New("no role ID found")
	}

	if spec.AWS.InstanceProfileName, ok = annotations[instanceProfileNameKey]; !ok {
		return nil, errors.New("no instance profile ID found")
	}

	if spec.AWS.PolicyName, ok = annotations[policyNameKey]; !ok {
		return nil, errors.New("no policy name found")
	}

	if spec.AWS.AvailabilityZone, ok = annotations[availabilityZoneKey]; !ok {
		return nil, errors.New("no availability zone found")
	}
	spec.AWS.SecurityGroupID, _ = annotations[securityGroupIDKey]

	return spec, nil
}

func (a *aws) userData(
	buf *bytes.Buffer,
	instanceName string,
	node *api.NodeSpec,
	clusterState *api.Cluster,
	dc provider.DatacenterMeta,
	key *api.KeyCert,
) error {
	data := ktemplate.Data{
		DC:                node.DatacenterName,
		ClusterName:       clusterState.Metadata.Name,
		SSHAuthorizedKeys: []string{},
		EtcdURL:           clusterState.Address.EtcdURL,
		APIServerURL:      clusterState.Address.URL,
		Region:            dc.Spec.AWS.Region,
		Name:              instanceName,
		ClientKey:         key.Key.Base64(),
		ClientCert:        key.Cert.Base64(),
		RootCACert:        clusterState.Status.RootCA.Cert.Base64(),
		ApiserverPubSSH:   clusterState.Status.ApiserverSSH,
		ApiserverToken:    clusterState.Address.Token,
		FlannelCIDR:       clusterState.Spec.Cloud.Network.Flannel.CIDR,
	}

	tpl, err := template.
		New("aws-cloud-config-node.yaml").
		Funcs(ktemplate.FuncMap).
		ParseFiles(tplPath)
	if err != nil {
		return fmt.Errorf("failed to parse cloud config: %v", err)
	}
	return tpl.Execute(buf, data)
}

func (a *aws) CreateNodes(ctx context.Context, cluster *api.Cluster, node *api.NodeSpec, num int) ([]*api.Node, error) {
	dc, ok := a.datacenters[node.DatacenterName]
	if !ok || dc.Spec.AWS == nil {
		return nil, fmt.Errorf("invalid datacenter %q", node.DatacenterName)
	}
	if node.AWS.Type == "" {
		return nil, errors.New("no AWS node type specified")
	}
	client, err := a.getEC2client(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed get ec2 client: %v", err)
	}
	var createdNodes []*api.Node
	var buf bytes.Buffer
	for i := 0; i < num; i++ {
		buf.Reset()
		id := provider.ShortUID(5)
		instanceName := fmt.Sprintf("kubermatic-%s-%s", cluster.Metadata.Name, id)

		clientKC, err := cluster.CreateKeyCert(instanceName, []string{})
		if err != nil {
			return createdNodes, fmt.Errorf("failed to create key cert: %v", err)
		}

		if err = a.userData(&buf, instanceName, node, cluster, dc, clientKC); err != nil {
			return createdNodes, fmt.Errorf("failed to generate user data: %v", err)
		}
		netSpec := []*ec2.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              sdk.Int64(0), // eth0
				AssociatePublicIpAddress: sdk.Bool(true),
				DeleteOnTermination:      sdk.Bool(true),
				SubnetId:                 sdk.String(cluster.Spec.Cloud.AWS.SubnetID),
			},
		}

		instanceRequest := &ec2.RunInstancesInput{
			ImageId:           sdk.String(dc.Spec.AWS.AMI),
			MaxCount:          sdk.Int64(1),
			MinCount:          sdk.Int64(1),
			InstanceType:      sdk.String(node.AWS.Type),
			UserData:          sdk.String(base64.StdEncoding.EncodeToString(buf.Bytes())),
			KeyName:           sdk.String(cluster.Spec.Cloud.AWS.SSHKeyName),
			NetworkInterfaces: netSpec,
			IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
				Name: sdk.String(fmt.Sprintf("kubermatic-instance-profile-%s", cluster.Metadata.Name)),
			},
		}

		newNode, err := launch(client, instanceName, instanceRequest, cluster)

		if err != nil {
			return createdNodes, fmt.Errorf("failed to launch node: %v", err)
		}
		createdNodes = append(createdNodes, newNode)
	}
	return createdNodes, nil
}

func (a *aws) Nodes(ctx context.Context, cluster *api.Cluster) ([]*api.Node, error) {
	client, err := a.getEC2client(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get ec2 client: %v", err)
	}

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{{
			// TODO: Direct Tag filtering
			Name: sdk.String(awsFilterState),
			Values: []*string{
				sdk.String(awsFilterRunning),
				sdk.String(awsFilterPending),
			},
		}},
	}

	resp, err := client.DescribeInstances(params)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %v", err)
	}

	nodes := make([]*api.Node, 0, len(resp.Reservations))
	for _, n := range resp.Reservations {
		for _, instance := range n.Instances {
			var isOwner bool
			for _, tag := range instance.Tags {
				if *tag.Key == defaultKubermaticClusterIDTagKey && *tag.Value == cluster.Metadata.UID {
					isOwner = true
				}
			}
			if isOwner {
				nodes = append(nodes, createNode(instance))
			}
		}
	}
	return nodes, nil
}

func (a *aws) DeleteNodes(ctx context.Context, cluster *api.Cluster, UIDs []string) error {
	client, err := a.getEC2client(cluster)
	if err != nil {
		return fmt.Errorf("failed to get ec2 client: %v", err)
	}

	awsInstanceIds := make([]*string, len(UIDs))
	for i := 0; i < len(UIDs); i++ {
		awsInstanceIds[i] = sdk.String(UIDs[i])
	}

	terminateRequest := &ec2.TerminateInstancesInput{
		InstanceIds: awsInstanceIds,
	}

	_, err = client.TerminateInstances(terminateRequest)
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %v", err)
	}
	return nil
}

func (a *aws) getSession(cluster *api.Cluster) (*session.Session, error) {
	awsSpec := cluster.Spec.Cloud.GetAWS()
	config := sdk.NewConfig()
	dc, found := a.datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found || dc.Spec.AWS == nil {
		return nil, fmt.Errorf("can't find datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}
	config = config.WithRegion(dc.Spec.AWS.Region)
	config = config.WithCredentials(credentials.NewStaticCredentials(awsSpec.AccessKeyID, awsSpec.SecretAccessKey, ""))
	// TODO: specify retrycount
	config = config.WithMaxRetries(3)
	return session.New(config), nil
}

func (a *aws) getEC2client(cluster *api.Cluster) (*ec2.EC2, error) {
	sess, err := a.getSession(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get aws session: %v", err)
	}
	return ec2.New(sess), nil
}

func (a *aws) getIAMclient(cluster *api.Cluster) (*iam.IAM, error) {
	sess, err := a.getSession(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get aws session: %v", err)
	}
	return iam.New(sess), nil
}

func createNode(instance *ec2.Instance) *api.Node {

	privateIP := ""
	publicIP := ""
	if instance.PrivateIpAddress != nil {
		privateIP = *instance.PrivateIpAddress
	}

	if instance.PublicIpAddress != nil {
		publicIP = *instance.PublicIpAddress
	}

	return &api.Node{
		Metadata: api.Metadata{
			UID:  *instance.InstanceId,
			Name: *instance.PrivateDnsName,
		},
		Status: api.NodeStatus{
			Addresses: api.NodeAddresses{
				Public:  publicIP,
				Private: privateIP,
			},
		},
		Spec: api.NodeSpec{
			DatacenterName: *instance.Placement.AvailabilityZone,
			AWS: &api.AWSNodeSpec{
				Type: *instance.InstanceType,
			},
		},
	}
}

func launch(client *ec2.EC2, name string, instance *ec2.RunInstancesInput, cluster *api.Cluster) (*api.Node, error) {
	serverReq, err := client.RunInstances(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to run instance: %v", err)
	}

	_, err = client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{serverReq.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   sdk.String(defaultKubermaticClusterIDTagKey),
				Value: sdk.String(cluster.Metadata.UID),
			},
			{
				Key:   sdk.String(defaultKubermaticClusterNameTagKey),
				Value: sdk.String(cluster.Metadata.Name),
			},
			{
				Key:   sdk.String(awsFilterName),
				Value: sdk.String(name),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tags: %v", err)
	}

	// Allow unchecked source/destination addresses for flannel
	_, err = client.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		SourceDestCheck: &ec2.AttributeBooleanValue{
			Value: sdk.Bool(true),
		},
		InstanceId: serverReq.Instances[0].InstanceId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to disable source/destination checking: %v", err)
	}

	// Change to our security group
	_, err = client.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		InstanceId: serverReq.Instances[0].InstanceId,
		Groups:     []*string{sdk.String(cluster.Spec.Cloud.AWS.SecurityGroupID)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach instance to security group group: %v", err)
	}

	return createNode(serverReq.Instances[0]), nil
}

func (a *aws) doCleanUpAWS(c *api.Cluster) error {
	client, err := a.getEC2client(c)
	if err != nil {
		return fmt.Errorf("failed to get ec2 client: %v", err)
	}

	// alive tests for living instances
	alive := func() (bool, error) {
		resp, err := client.DescribeInstances(&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{{
				Name:   sdk.String("tag-value"),
				Values: []*string{sdk.String(c.Metadata.UID)},
			},
				{
					Name: sdk.String("instance-state-name"),
					Values: []*string{
						sdk.String(ec2.InstanceStateNamePending),
						sdk.String(ec2.InstanceStateNameRunning),
						sdk.String(ec2.InstanceStateNameShuttingDown),
						sdk.String(ec2.InstanceStateNameStopping),
						sdk.String(ec2.InstanceStateNameStopped),
					},
				}},
		})
		if err != nil {
			return true, err
		}

		// Look for living instaces
		for _, reservation := range resp.Reservations {
			if len(reservation.Instances) > 0 {
				return true, nil
			}
		}
		return false, nil
	}

	// Wait for nodes to terminate
	for {
		instancesAlive, err := alive()
		if err != nil {
			return err
		}
		if !instancesAlive {
			break
		}
		time.Sleep(time.Second * 45)
	}

	if c.Spec.Cloud.AWS.SecurityGroupID != "" {
		_, err = client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: sdk.String(c.Spec.Cloud.AWS.SecurityGroupID),
		})
		if err != nil {
			glog.V(2).Infof("Failed to delete security group %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.SecurityGroupID, c.Metadata.Name, err)
		}
	}

	if c.Spec.Cloud.AWS.InitMode == api.AWSInitCreateVpc {
		if c.Spec.Cloud.AWS.RouteTableID != "" {
			_, err = client.DeleteRouteTable(&ec2.DeleteRouteTableInput{
				RouteTableId: sdk.String(c.Spec.Cloud.AWS.RouteTableID),
			})
			if err != nil {
				glog.V(2).Infof("Failed to delete RouteTable %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.RouteTableID, c.Metadata.Name, err)
			}
		}

		if c.Spec.Cloud.AWS.InternetGatewayID != "" && c.Spec.Cloud.AWS.VPCId != "" {
			_, err = client.DeleteSubnet(&ec2.DeleteSubnetInput{
				SubnetId: sdk.String(c.Spec.Cloud.AWS.SubnetID),
			})
			if err != nil {
				glog.V(2).Infof("Failed to delete Subnet %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.SubnetID, c.Metadata.Name, err)
			}
		}

		if c.Spec.Cloud.AWS.InternetGatewayID != "" {
			_, err = client.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{
				InternetGatewayId: sdk.String(c.Spec.Cloud.AWS.InternetGatewayID),
			})
			if err != nil {
				glog.V(2).Infof("Failed to delete InternetGateway %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.InternetGatewayID, c.Metadata.Name, err)
			}
		}

		if c.Spec.Cloud.AWS.VPCId != "" {
			_, err = client.DeleteVpc(&ec2.DeleteVpcInput{
				VpcId: sdk.String(c.Spec.Cloud.AWS.VPCId),
			})
			if err != nil {
				glog.V(2).Infof("Failed to delete VPC %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.VPCId, c.Metadata.Name, err)
			}
		}
	}

	svcIAM, err := a.getIAMclient(c)
	if err != nil {
		return err
	}

	if c.Spec.Cloud.AWS.RoleName != "" && c.Spec.Cloud.AWS.InstanceProfileName != "" {
		_, err := svcIAM.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
			RoleName:            sdk.String(c.Spec.Cloud.AWS.RoleName),
			InstanceProfileName: sdk.String(c.Spec.Cloud.AWS.InstanceProfileName),
		})
		if err != nil {
			glog.V(2).Infof("Failed to remove role %s from instance profile %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.RoleName, c.Spec.Cloud.AWS.InstanceProfileName, c.Metadata.Name, err)
		}
	}

	if c.Spec.Cloud.AWS.InstanceProfileName != "" {
		_, err := svcIAM.DeleteInstanceProfile(&iam.DeleteInstanceProfileInput{InstanceProfileName: &c.Spec.Cloud.AWS.InstanceProfileName})
		if err != nil {
			glog.V(2).Infof("Failed to delete InstanceProfile %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.InstanceProfileName, c.Metadata.Name, err)
		}
	}
	if c.Spec.Cloud.AWS.RoleName != "" && c.Spec.Cloud.AWS.PolicyName != "" {
		_, err := svcIAM.DetachRolePolicy(&iam.DetachRolePolicyInput{
			RoleName:  sdk.String(c.Spec.Cloud.AWS.RoleName),
			PolicyArn: sdk.String(c.Spec.Cloud.AWS.PolicyName),
		})
		if err != nil {
			glog.V(2).Infof("Failed to detach role policy %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.PolicyName, c.Metadata.Name, err)
		}
	}

	if c.Spec.Cloud.AWS.RoleName != "" {
		_, err := svcIAM.DeleteRole(&iam.DeleteRoleInput{RoleName: &c.Spec.Cloud.AWS.RoleName})
		if err != nil {
			glog.V(2).Infof("Failed to delete Role %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.RoleName, c.Metadata.Name, err)
		}
	}

	if c.Spec.Cloud.AWS.PolicyName != "" {
		_, err := svcIAM.DeletePolicy(&iam.DeletePolicyInput{
			PolicyArn: sdk.String(c.Spec.Cloud.AWS.PolicyName),
		})
		if err != nil {
			glog.V(2).Infof("Failed to delete role policy %s during aws-cleanup for cluster %s : %v", c.Spec.Cloud.AWS.PolicyName, c.Metadata.Name, err)
		}
	}

	return nil
}

func (a *aws) CleanUp(c *api.Cluster) error {
	go func() { _ = a.doCleanUpAWS(c) }()
	return nil
}
