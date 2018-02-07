package aws

import (
	"crypto/md5"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	machinessh "github.com/kubermatic/machine-controller/pkg/ssh"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
)

type provider struct {
	privateKey *machinessh.PrivateKey
}

// New returns a aws provider
func New(privateKey *machinessh.PrivateKey) cloud.Provider {
	return &provider{privateKey: privateKey}
}

const (
	nameTag       = "Name"
	machineUIDTag = "Machine-UID"

	policyRoute53FullAccess = "arn:aws:iam::aws:policy/AmazonRoute53FullAccess"
	policyEC2FullAccess     = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"

	defaultRoleName            = "kubernetes-v1"
	defaultInstanceProfileName = "kubernetes-v1"
	defaultSecurityGroupName   = "kubernetes-v1"
)

var (
	volumeTypes = sets.NewString(
		ec2.VolumeTypeStandard,
		ec2.VolumeTypeIo1,
		ec2.VolumeTypeGp2,
		ec2.VolumeTypeSc1,
		ec2.VolumeTypeSt1,
	)

	roleARNS = []string{policyRoute53FullAccess, policyEC2FullAccess}

	instanceProfileRole = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "ec2.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }
  ]
}`

	amis = map[providerconfig.OperatingSystem]map[string]string{
		providerconfig.OperatingSystemCoreos: {
			"ap-northeast-1": "ami-6a6bec0c",
			"ap-northeast-2": "ami-7fb41211",
			"ap-south-1":     "ami-02b4fd6d",
			"ap-southeast-1": "ami-cb096db7",
			"ap-southeast-2": "ami-7957a31b",
			"ca-central-1":   "ami-9c16adf8",
			"cn-north-1":     "ami-e803d185",
			"eu-central-1":   "ami-31c74e5e",
			"eu-west-1":      "ami-c8a811b1",
			"eu-west-2":      "ami-8ccdd3e8",
			"sa-east-1":      "ami-af84c3c3",
			"us-east-1":      "ami-6dfb9a17",
			"us-east-2":      "ami-01e2cb64",
			"us-gov-west-1":  "ami-6bad220a",
			"us-west-1":      "ami-7d81bb1d",
			"us-west-2":      "ami-c167bdb9",
		},
		providerconfig.OperatingSystemUbuntu: {
			"ap-northeast-1": "ami-42ca4724",
			"ap-south-1":     "ami-84dc94eb",
			"ap-southeast-1": "ami-29aece55",
			"ca-central-1":   "ami-b0c67cd4",
			"eu-central-1":   "ami-13b8337c",
			"eu-west-1":      "ami-63b0341a",
			"sa-east-1":      "ami-8181c7ed",
			"us-east-1":      "ami-3dec9947",
			"us-west-1":      "ami-1a17137a",
			"cn-north-1":     "ami-fc25f791",
			"cn-northwest-1": "ami-e5b0a587",
			"us-gov-west-1":  "ami-6261ee03",
			"ap-northeast-2": "ami-5027813e",
			"ap-southeast-2": "ami-9b8076f9",
			"eu-west-2":      "ami-22415846",
			"us-east-2":      "ami-597d553c",
			"us-west-2":      "ami-a2e544da",
			"eu-west-3":      "ami-794bfc04",
		},
	}

	publicKeyCreationLock = &sync.Mutex{}
)

type Config struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`

	Region           string `json:"region"`
	AvailabilityZone string `json:"availabilityZone"`

	VpcID            string   `json:"vpcId"`
	SubnetID         string   `json:"subnetId"`
	SecurityGroupIDs []string `json:"securityGroupIDs"`
	InstanceProfile  string   `json:"instanceProfile"`

	InstanceType string            `json:"instanceType"`
	AMI          string            `json:"ami"`
	DiskSize     int64             `json:"diskSize"`
	DiskType     string            `json:"diskType"`
	Tags         map[string]string `json:"tags"`
}

func getDefaultAMIID(os providerconfig.OperatingSystem, region string) (string, error) {
	amis, osSupported := amis[os]
	if !osSupported {
		return "", fmt.Errorf("operating system %q not supported", os)
	}

	id, regionFound := amis[region]
	if !regionFound {
		return "", fmt.Errorf("specified region %q not supported with this operating system %q", region, os)
	}

	return id, nil
}

func getConfig(s runtime.RawExtension) (*Config, *providerconfig.Config, error) {
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Raw, &pconfig)
	if err != nil {
		return nil, nil, err
	}
	c := Config{}
	err = json.Unmarshal(pconfig.CloudProviderSpec.Raw, &c)
	return &c, &pconfig, err
}

func getSession(id, secret, token, region string) (*session.Session, error) {
	config := aws.NewConfig()
	config = config.WithRegion(region)
	config = config.WithCredentials(credentials.NewStaticCredentials(id, secret, token))
	config = config.WithMaxRetries(3)
	return session.NewSession(config)
}

func getIAMclient(id, secret, region string) (*iam.IAM, error) {
	sess, err := getSession(id, secret, "", region)
	if err != nil {
		return nil, fmt.Errorf("failed to get aws session: %v", err)
	}
	return iam.New(sess), nil
}

func getEC2client(id, secret, region string) (*ec2.EC2, error) {
	sess, err := getSession(id, secret, "", region)
	if err != nil {
		return nil, fmt.Errorf("failed to get aws session: %v", err)
	}
	return ec2.New(sess), nil
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, bool, error) {
	return spec, false, nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	config, pc, err := getConfig(spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	ec2Client, err := getEC2client(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return fmt.Errorf("failed to create ec2 client: %v", err)
	}
	if config.AMI != "" {
		_, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{
			ImageIds: aws.StringSlice([]string{config.AMI}),
		})
		if err != nil {
			return fmt.Errorf("failed to validate ami: %v", err)
		}
	} else {
		_, err := getDefaultAMIID(pc.OperatingSystem, config.Region)
		if err != nil {
			return fmt.Errorf("invalid region+os configuration: %v", err)
		}
	}

	if _, err := getVpc(ec2Client, config.VpcID); err != nil {
		return fmt.Errorf("invalid vpc %q specified: %v", config.VpcID, err)
	}

	_, err = ec2Client.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{ZoneNames: aws.StringSlice([]string{config.AvailabilityZone})})
	if err != nil {
		return fmt.Errorf("invalid zone %q specified: %v", config.AvailabilityZone, err)
	}

	_, err = ec2Client.DescribeRegions(&ec2.DescribeRegionsInput{RegionNames: aws.StringSlice([]string{config.Region})})
	if err != nil {
		return fmt.Errorf("invalid region %q specified: %v", config.Region, err)
	}

	if len(config.SecurityGroupIDs) > 0 {
		_, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			GroupIds: aws.StringSlice(config.SecurityGroupIDs),
		})
		if err != nil {
			return fmt.Errorf("failed to validate security group id's: %v", err)
		}
	}

	iamClient, err := getIAMclient(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return fmt.Errorf("failed to create iam client: %v", err)
	}

	if config.InstanceProfile != "" {
		_, err := iamClient.GetInstanceProfile(&iam.GetInstanceProfileInput{InstanceProfileName: aws.String(config.InstanceProfile)})
		if err != nil {
			return fmt.Errorf("failed to validate instance profile: %v", err)
		}
	}

	if !volumeTypes.Has(config.DiskType) {
		return fmt.Errorf("invalid volume type %s specified. Supported: %s", config.DiskType, volumeTypes)
	}

	return nil
}

func getVpc(client *ec2.EC2, id string) (*ec2.Vpc, error) {
	vpcOut, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(id)}},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list vpc's: %v", err)
	}

	if len(vpcOut.Vpcs) != 1 {
		return nil, fmt.Errorf("unable to find specified vpc with id %q", id)
	}

	return vpcOut.Vpcs[0], nil
}

func ensureDefaultSecurityGroupExists(client *ec2.EC2, vpc *ec2.Vpc) (string, error) {
	sgOut, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupNames: aws.StringSlice([]string{defaultSecurityGroupName}),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "InvalidGroup.NotFound" {
				glog.V(4).Infof("creating security group %s...", defaultSecurityGroupName)
				csgOut, err := client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
					VpcId:       vpc.VpcId,
					GroupName:   aws.String(defaultSecurityGroupName),
					Description: aws.String("Kubernetes security group"),
				})
				if err != nil {
					return "", fmt.Errorf("failed to create security group: %v", err)
				}
				groupID := aws.StringValue(csgOut.GroupId)

				// Allow SSH from everywhere
				_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
					CidrIp:     aws.String("0.0.0.0/0"),
					FromPort:   aws.Int64(22),
					ToPort:     aws.Int64(22),
					GroupId:    csgOut.GroupId,
					IpProtocol: aws.String("tcp"),
				})
				if err != nil {
					return "", fmt.Errorf("failed to authorize security group ingress rule for ssh to security group %s: %v", groupID, err)
				}

				// Allow kubelet 10250 from everywhere
				_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
					CidrIp:     aws.String("0.0.0.0/0"),
					FromPort:   aws.Int64(10250),
					ToPort:     aws.Int64(10250),
					GroupId:    csgOut.GroupId,
					IpProtocol: aws.String("tcp"),
				})
				if err != nil {
					return "", fmt.Errorf("failed to authorize security group ingress rule for kubelet port 10250 to security group %s: %v", groupID, err)
				}

				// Allow node-to-node communication
				_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
					SourceSecurityGroupName: aws.String(defaultSecurityGroupName),
					GroupId:                 csgOut.GroupId,
				})
				if err != nil {
					return "", fmt.Errorf("failed to authorize security group ingress rule for node-to-node communication to security group %s: %v", groupID, err)
				}

				glog.V(4).Infof("security group %s successfully created", defaultSecurityGroupName)
				return groupID, nil
			}
		}
		return "", fmt.Errorf("failed to list security group: %v", err)
	}

	glog.V(6).Infof("security group %s already exists", defaultSecurityGroupName)
	return aws.StringValue(sgOut.SecurityGroups[0].GroupId), nil
}

func ensureSSHKeysExist(client *ec2.EC2, key *machinessh.PrivateKey) (string, error) {
	publicKeyCreationLock.Lock()
	defer publicKeyCreationLock.Unlock()

	publicKey := key.PublicKey()
	out, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key to PKIX format: %v", err)
	}

	//Basically copied from golang.org/x/crypto/ssh/keys.go:1013
	md5sum := md5.Sum(out)
	hexarray := make([]string, len(md5sum))
	for i, c := range md5sum {
		hexarray[i] = hex.EncodeToString([]byte{c})
	}
	fingerprint := strings.Join(hexarray, ":")

	keyout, err := client.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("fingerprint"),
				Values: aws.StringSlice([]string{fingerprint}),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to list public keys from aws: %v", err)
	}
	if len(keyout.KeyPairs) != 0 {
		glog.V(6).Infof("ssh public key already exists")
		return *keyout.KeyPairs[0].KeyName, nil
	}

	glog.V(4).Infof("importing ssh public key into aws...")

	spk, err := ssh.NewPublicKey(&publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %v", err)
	}

	importOut, err := client.ImportKeyPair(&ec2.ImportKeyPairInput{
		KeyName:           aws.String(key.Name()),
		PublicKeyMaterial: ssh.MarshalAuthorizedKey(spk),
	})
	if err != nil {
		return "", fmt.Errorf("failed to import public key at aws: %v", err)
	}
	glog.V(4).Infof("successfully imported ssh public key into aws")
	return *importOut.KeyName, nil
}

func ensureDefaultRoleExists(client *iam.IAM) error {
	_, err := client.GetRole(&iam.GetRoleInput{RoleName: aws.String(defaultRoleName)})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchEntity" {
				glog.V(4).Infof("creating machine iam role %s...", defaultRoleName)
				paramsRole := &iam.CreateRoleInput{
					AssumeRolePolicyDocument: aws.String(instanceProfileRole),
					RoleName:                 aws.String(defaultRoleName),
				}
				_, err := client.CreateRole(paramsRole)
				if err != nil {
					return fmt.Errorf("failed to create role: %v", err)
				}

				for _, arn := range roleARNS {
					paramsAttachPolicy := &iam.AttachRolePolicyInput{
						PolicyArn: aws.String(arn),
						RoleName:  aws.String(defaultRoleName),
					}
					_, err = client.AttachRolePolicy(paramsAttachPolicy)
					if err != nil {
						return fmt.Errorf("failed to attach role %q to policy %q: %v", defaultRoleName, arn, err)
					}
				}
				glog.V(4).Infof("machine iam role %s successfully created", defaultRoleName)
				return nil
			}
			return fmt.Errorf("failed to get role %s: %s - %s", defaultRoleName, awsErr.Code(), awsErr.Message())
		}
		return fmt.Errorf("failed to get role %s: %v", defaultRoleName, err)
	}
	glog.V(6).Infof("machine iam role %s already exists", defaultRoleName)
	return nil
}

func ensureDefaultInstanceProfileExists(client *iam.IAM) error {
	err := ensureDefaultRoleExists(client)
	if err != nil {
		return fmt.Errorf("failed to ensure that role %q exists: %v", defaultRoleName, err)
	}

	_, err = client.GetInstanceProfile(&iam.GetInstanceProfileInput{InstanceProfileName: aws.String(defaultInstanceProfileName)})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchEntity" {
				glog.V(4).Infof("creating instance profile %s...", defaultInstanceProfileName)
				paramsInstanceProfile := &iam.CreateInstanceProfileInput{
					InstanceProfileName: aws.String(defaultInstanceProfileName),
				}
				_, err = client.CreateInstanceProfile(paramsInstanceProfile)
				if err != nil {
					return fmt.Errorf("failed to create instance profile: %v", err)
				}

				paramsAddRole := &iam.AddRoleToInstanceProfileInput{
					InstanceProfileName: aws.String(defaultInstanceProfileName),
					RoleName:            aws.String(defaultRoleName),
				}
				_, err = client.AddRoleToInstanceProfile(paramsAddRole)
				if err != nil {
					return fmt.Errorf("failed to add role %q to instance profile %q: %v", defaultInstanceProfileName, defaultRoleName, err)
				}
				glog.V(4).Infof("instance profile %s successfully created", defaultInstanceProfileName)
				return nil
			}
			return fmt.Errorf("failed to get instance profile %s: %s - %s", defaultInstanceProfileName, awsErr.Code(), awsErr.Message())
		}
		return fmt.Errorf("failed to get instance profile: %v", err)
	}
	glog.V(6).Infof("instance profile %s already exists", defaultInstanceProfileName)

	return nil
}

func (p *provider) Create(machine *v1alpha1.Machine, userdata string) (instance.Instance, error) {
	config, pc, err := getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	ec2Client, err := getEC2client(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to create ec2 client: %v", err)
	}

	iamClient, err := getIAMclient(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to create iam client: %v", err)
	}

	instanceProfileName := config.InstanceProfile
	if instanceProfileName == "" {
		err = ensureDefaultInstanceProfileExists(iamClient)
		if err != nil {
			return nil, fmt.Errorf("failed ensure that the instance profile exists: %v", err)
		}
		instanceProfileName = defaultInstanceProfileName
	}

	vpc, err := getVpc(ec2Client, config.VpcID)
	if err != nil {
		return nil, fmt.Errorf("failed get vpc %s: %v", config.VpcID, err)
	}

	securityGroupIDs := config.SecurityGroupIDs
	if len(securityGroupIDs) == 0 {
		sgID, err := ensureDefaultSecurityGroupExists(ec2Client, vpc)
		if err != nil {
			return nil, fmt.Errorf("failed ensure that the security group exists: %v", err)
		}
		securityGroupIDs = append(securityGroupIDs, sgID)
	} else {

	}

	keyName, err := ensureSSHKeysExist(ec2Client, p.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed ensure that the ssh key '%s' exists: %v", p.privateKey.Name(), err)
	}

	amiID := config.AMI
	if amiID == "" {
		if amiID, err = getDefaultAMIID(pc.OperatingSystem, config.Region); err != nil {
			if err != nil {
				return nil, fmt.Errorf("invalid region+os configuration: %v", err)
			}
		}
	}

	tags := []*ec2.Tag{
		{
			Key:   aws.String(nameTag),
			Value: aws.String(machine.Spec.Name),
		},
		{
			Key:   aws.String(machineUIDTag),
			Value: aws.String(string(machine.UID)),
		},
	}

	for k, v := range config.Tags {
		tags = append(tags, &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	instanceRequest := &ec2.RunInstancesInput{
		ImageId: aws.String(amiID),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/xvda"),
				Ebs: &ec2.EbsBlockDevice{
					VolumeSize:          aws.Int64(config.DiskSize),
					DeleteOnTermination: aws.Bool(true),
					VolumeType:          aws.String(config.DiskType),
				},
			},
		},
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
		InstanceType: aws.String(config.InstanceType),
		UserData:     aws.String(base64.StdEncoding.EncodeToString([]byte(userdata))),
		KeyName:      aws.String(keyName),
		Placement: &ec2.Placement{
			AvailabilityZone: aws.String(config.AvailabilityZone),
		},
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int64(0), // eth0
				AssociatePublicIpAddress: aws.Bool(true),
				DeleteOnTermination:      aws.Bool(true),
				SubnetId:                 aws.String(config.SubnetID),
			},
		},
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String(instanceProfileName),
		},
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String(ec2.ResourceTypeInstance),
				Tags:         tags,
			},
		},
	}

	runOut, err := ec2Client.RunInstances(instanceRequest)
	if err != nil {
		return nil, fmt.Errorf("failed create instance at aws: %v", err)
	}
	awsInstance := &awsInstance{instance: runOut.Instances[0]}

	// Change to our security group
	_, err = ec2Client.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		InstanceId: runOut.Instances[0].InstanceId,
		Groups:     aws.StringSlice(securityGroupIDs),
	})
	if err != nil {
		delErr := p.Delete(machine)
		if delErr != nil {
			return nil, fmt.Errorf("failed to attach instance %s to security group %s & delete the created instance: %v", aws.StringValue(runOut.Instances[0].InstanceId), defaultSecurityGroupName, err)
		}
		return nil, fmt.Errorf("failed to attach instance %s to security group %s: %v", aws.StringValue(runOut.Instances[0].InstanceId), defaultSecurityGroupName, err)
	}

	return awsInstance, nil
}

func (p *provider) Delete(machine *v1alpha1.Machine) error {
	i, err := p.Get(machine)
	if err != nil {
		return fmt.Errorf("failed to get instance from aws: %v", err)
	}

	config, _, err := getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	ec2Client, err := getEC2client(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return fmt.Errorf("failed to create ec2 client: %v", err)
	}

	tOut, err := ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice([]string{i.ID()}),
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %v", err)
	}

	if *tOut.TerminatingInstances[0].PreviousState.Name != *tOut.TerminatingInstances[0].CurrentState.Name {
		glog.V(4).Infof("successfully triggered termination of instance %s at aws", i.ID())
	}

	return nil
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {
	config, _, err := getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	ec2Client, err := getEC2client(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to create ec2 client: %v", err)
	}

	inOut, err := ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:" + machineUIDTag),
				Values: aws.StringSlice([]string{string(machine.UID)}),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list instances from aws: %v", err)
	}

	if len(inOut.Reservations) == 0 || len(inOut.Reservations[0].Instances) == 0 {
		return nil, cloudprovidererrors.ErrInstanceNotFound
	}

	return &awsInstance{
		instance: inOut.Reservations[0].Instances[0],
	}, nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	return "", "aws", nil
}

type awsInstance struct {
	instance *ec2.Instance
}

func (d *awsInstance) Name() string {
	return getTagValue(nameTag, d.instance.Tags)
}

func (d *awsInstance) ID() string {
	return aws.StringValue(d.instance.InstanceId)
}

func (d *awsInstance) Addresses() []string {
	return []string{
		aws.StringValue(d.instance.PublicIpAddress),
		aws.StringValue(d.instance.PublicDnsName),
		aws.StringValue(d.instance.PrivateIpAddress),
		aws.StringValue(d.instance.PrivateDnsName),
	}
}

func (d *awsInstance) Status() instance.Status {
	switch *d.instance.State.Name {
	case ec2.InstanceStateNameRunning:
		return instance.StatusRunning
	case ec2.InstanceStateNamePending:
		return instance.StatusCreating
	case ec2.InstanceStateNameTerminated:
		return instance.StatusDeleted
	case ec2.InstanceStateNameShuttingDown:
		return instance.StatusDeleting
	default:
		return instance.StatusUnknown
	}
}

func getTagValue(name string, tags []*ec2.Tag) string {
	for _, t := range tags {
		if *t.Key == name {
			return *t.Value
		}
	}
	return ""
}
