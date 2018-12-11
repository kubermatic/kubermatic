package aws

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	"github.com/kubermatic/machine-controller/pkg/userdata/convert"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/golang/glog"
	gocache "github.com/patrickmn/go-cache"

	"sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type provider struct {
	configVarResolver *providerconfig.ConfigVarResolver
}

// New returns a aws provider
func New(configVarResolver *providerconfig.ConfigVarResolver) cloud.Provider {
	return &provider{configVarResolver: configVarResolver}
}

const (
	nameTag       = "Name"
	machineUIDTag = "Machine-UID"

	maxRetries = 100
)

var (
	volumeTypes = sets.NewString(
		ec2.VolumeTypeStandard,
		ec2.VolumeTypeIo1,
		ec2.VolumeTypeGp2,
		ec2.VolumeTypeSc1,
		ec2.VolumeTypeSt1,
	)

	amiFilters = map[providerconfig.OperatingSystem]amiFilter{
		providerconfig.OperatingSystemCoreos: {
			description: "CoreOS Container Linux stable*",
			// The AWS marketplace ID from CoreOS
			owner: "595879546273",
		},
		providerconfig.OperatingSystemCentOS: {
			description: "CentOS Linux 7 x86_64 HVM EBS*",
			// The AWS marketplace ID from AWS
			owner: "679593333241",
		},
		providerconfig.OperatingSystemUbuntu: {
			// Be as precise as possible - otherwise we might get a nightly dev build
			description: "Canonical, Ubuntu, 18.04 LTS, amd64 bionic image build on ????-??-??",
			// The AWS marketplace ID from Canonical
			owner: "099720109477",
		},
	}

	// cacheLock protects concurrent cache misses against a single key. This usually happens when multiple machines get created simultaneously
	// We lock so the first access updates/writes the data to the cache and afterwards everyone reads the cached data
	cacheLock = &sync.Mutex{}
	cache     = gocache.New(5*time.Minute, 5*time.Minute)
)

type RawConfig struct {
	AccessKeyID     providerconfig.ConfigVarString `json:"accessKeyId"`
	SecretAccessKey providerconfig.ConfigVarString `json:"secretAccessKey"`

	Region           providerconfig.ConfigVarString `json:"region"`
	AvailabilityZone providerconfig.ConfigVarString `json:"availabilityZone"`

	VpcID            providerconfig.ConfigVarString   `json:"vpcId"`
	SubnetID         providerconfig.ConfigVarString   `json:"subnetId"`
	SecurityGroupIDs []providerconfig.ConfigVarString `json:"securityGroupIDs"`
	InstanceProfile  providerconfig.ConfigVarString   `json:"instanceProfile"`

	InstanceType providerconfig.ConfigVarString `json:"instanceType"`
	AMI          providerconfig.ConfigVarString `json:"ami"`
	DiskSize     int64                          `json:"diskSize"`
	DiskType     providerconfig.ConfigVarString `json:"diskType"`
	Tags         map[string]string              `json:"tags"`
}

type Config struct {
	AccessKeyID     string
	SecretAccessKey string

	Region           string
	AvailabilityZone string

	VpcID            string
	SubnetID         string
	SecurityGroupIDs []string
	InstanceProfile  string

	InstanceType string
	AMI          string
	DiskSize     int64
	DiskType     string
	Tags         map[string]string
}

type amiFilter struct {
	description string
	owner       string
}

func getDefaultAMIID(client *ec2.EC2, os providerconfig.OperatingSystem, region string) (string, error) {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	filter, osSupported := amiFilters[os]
	if !osSupported {
		return "", fmt.Errorf("operating system %q not supported", os)
	}

	cacheKey := fmt.Sprintf("ami-id-%s-%s", region, os)
	amiID, found := cache.Get(cacheKey)
	if found {
		glog.V(4).Info("found AMI-ID in cache!")
		return amiID.(string), nil
	}

	imagesOut, err := client.DescribeImages(&ec2.DescribeImagesInput{
		Owners: aws.StringSlice([]string{filter.owner}),
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("description"),
				Values: aws.StringSlice([]string{filter.description}),
			},
			{
				Name:   aws.String("virtualization-type"),
				Values: aws.StringSlice([]string{"hvm"}),
			},
			{
				Name:   aws.String("root-device-type"),
				Values: aws.StringSlice([]string{"ebs"}),
			},
		},
	})
	if err != nil {
		return "", err
	}

	if len(imagesOut.Images) == 0 {
		return "", fmt.Errorf("could not find Image for '%s'", os)
	}

	image := imagesOut.Images[0]
	for _, v := range imagesOut.Images {
		itime, _ := time.Parse(time.RFC3339, *image.CreationDate)
		vtime, _ := time.Parse(time.RFC3339, *v.CreationDate)
		if vtime.After(itime) {
			image = v
		}
	}

	cache.SetDefault(cacheKey, *image.ImageId)
	return *image.ImageId, nil
}

func getDefaultRootDevicePath(os providerconfig.OperatingSystem) (string, error) {
	switch os {
	case providerconfig.OperatingSystemUbuntu:
		return "/dev/sda1", nil
	case providerconfig.OperatingSystemCentOS:
		return "/dev/sda1", nil
	case providerconfig.OperatingSystemCoreos:
		return "/dev/xvda", nil
	}

	return "", fmt.Errorf("no default root path found for %s operating system", os)
}

func (p *provider) getConfig(s v1alpha1.ProviderConfig) (*Config, *providerconfig.Config, error) {
	if s.Value == nil {
		return nil, nil, fmt.Errorf("machine.spec.providerconfig.value is nil")
	}
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Value.Raw, &pconfig)
	if err != nil {
		return nil, nil, err
	}
	rawConfig := RawConfig{}
	if err := json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal: %v", err)
	}
	c := Config{}
	c.AccessKeyID, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.AccessKeyID, "AWS_ACCESS_KEY_ID")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"accessKeyId\" field, error = %v", err)
	}
	c.SecretAccessKey, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.SecretAccessKey, "AWS_SECRET_ACCESS_KEY")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"secretAccessKey\" field, error = %v", err)
	}
	c.Region, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Region)
	if err != nil {
		return nil, nil, err
	}
	c.VpcID, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.VpcID)
	if err != nil {
		return nil, nil, err
	}
	c.SubnetID, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.SubnetID)
	if err != nil {
		return nil, nil, err
	}
	c.AvailabilityZone, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.AvailabilityZone)
	if err != nil {
		return nil, nil, err
	}
	for _, securityGroupIDRaw := range rawConfig.SecurityGroupIDs {
		securityGroupID, err := p.configVarResolver.GetConfigVarStringValue(securityGroupIDRaw)
		if err != nil {
			return nil, nil, err
		}
		c.SecurityGroupIDs = append(c.SecurityGroupIDs, securityGroupID)
	}
	c.InstanceProfile, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.InstanceProfile)
	if err != nil {
		return nil, nil, err
	}
	c.InstanceType, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.InstanceType)
	if err != nil {
		return nil, nil, err
	}
	c.AMI, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.AMI)
	if err != nil {
		return nil, nil, err
	}
	c.DiskSize = rawConfig.DiskSize
	c.DiskType, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.DiskType)
	if err != nil {
		return nil, nil, err
	}
	c.Tags = rawConfig.Tags

	return &c, &pconfig, err
}

func getSession(id, secret, token, region string) (*session.Session, error) {
	config := aws.NewConfig()
	config = config.WithRegion(region)
	config = config.WithCredentials(credentials.NewStaticCredentials(id, secret, token))
	config = config.WithMaxRetries(maxRetries)
	return session.NewSession(config)
}

func getIAMclient(id, secret, region string) (*iam.IAM, error) {
	sess, err := getSession(id, secret, "", region)
	if err != nil {
		return nil, awsErrorToTerminalError(err, "failed to get aws session")
	}
	return iam.New(sess), nil
}

func getEC2client(id, secret, region string) (*ec2.EC2, error) {
	sess, err := getSession(id, secret, "", region)
	if err != nil {
		return nil, awsErrorToTerminalError(err, "failed to get aws session")
	}
	return ec2.New(sess), nil
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, error) {
	return spec, nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	config, pc, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	if _, osSupported := amiFilters[pc.OperatingSystem]; !osSupported {
		return fmt.Errorf("unsupported os %s", pc.OperatingSystem)
	}

	if !volumeTypes.Has(config.DiskType) {
		return fmt.Errorf("invalid volume type %s specified. Supported: %s", config.DiskType, volumeTypes)
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

	if len(config.SecurityGroupIDs) == 0 {
		return errors.New("no security groups were specified")
	}
	_, err = ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: aws.StringSlice(config.SecurityGroupIDs),
	})
	if err != nil {
		return fmt.Errorf("failed to validate security group id's: %v", err)
	}

	iamClient, err := getIAMclient(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return fmt.Errorf("failed to create iam client: %v", err)
	}

	if config.InstanceProfile == "" {
		return fmt.Errorf("invalid instance profile specified %q: %v", config.InstanceProfile, err)
	}
	if _, err := iamClient.GetInstanceProfile(&iam.GetInstanceProfileInput{InstanceProfileName: aws.String(config.InstanceProfile)}); err != nil {
		return fmt.Errorf("failed to validate instance profile: %v", err)
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
		return nil, awsErrorToTerminalError(err, "failed to list vpc's")
	}

	if len(vpcOut.Vpcs) != 1 {
		return nil, fmt.Errorf("unable to find specified vpc with id %q", id)
	}

	return vpcOut.Vpcs[0], nil
}

func (p *provider) Create(machine *v1alpha1.Machine, data *cloud.MachineCreateDeleteData, userdata string) (instance.Instance, error) {
	config, pc, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	ec2Client, err := getEC2client(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return nil, err
	}

	rootDevicePath, err := getDefaultRootDevicePath(pc.OperatingSystem)
	if err != nil {
		return nil, err
	}

	amiID := config.AMI
	if amiID == "" {
		if amiID, err = getDefaultAMIID(ec2Client, pc.OperatingSystem, config.Region); err != nil {
			if err != nil {
				return nil, cloudprovidererrors.TerminalError{
					Reason:  common.InvalidConfigurationMachineError,
					Message: fmt.Sprintf("Invalid Region and Operating System configuration: %v", err),
				}
			}
		}
	}

	if pc.OperatingSystem != providerconfig.OperatingSystemCoreos {
		// Gzip the userdata in case we don't use CoreOS.
		userdata, err = convert.GzipString(userdata)
		if err != nil {
			return nil, fmt.Errorf("failed to gzip the userdata")
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
				DeviceName: aws.String(rootDevicePath),
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
			Name: aws.String(config.InstanceProfile),
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
		return nil, awsErrorToTerminalError(err, "failed create instance at aws")
	}
	awsInstance := &awsInstance{instance: runOut.Instances[0]}

	// Change to our security group
	_, modifyInstanceErr := ec2Client.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		InstanceId: runOut.Instances[0].InstanceId,
		Groups:     aws.StringSlice(config.SecurityGroupIDs),
	})
	if modifyInstanceErr != nil {
		_, err := ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{
			InstanceIds: []*string{runOut.Instances[0].InstanceId},
		})
		if err != nil {
			return nil, awsErrorToTerminalError(modifyInstanceErr, fmt.Sprintf("failed to delete instance %s due to %v after attaching to security groups %v", aws.StringValue(runOut.Instances[0].InstanceId), err, config.SecurityGroupIDs))
		}
		return nil, awsErrorToTerminalError(modifyInstanceErr, fmt.Sprintf("failed to attach instance %s to security group %v", aws.StringValue(runOut.Instances[0].InstanceId), config.SecurityGroupIDs))
	}

	return awsInstance, nil
}

func (p *provider) Cleanup(machine *v1alpha1.Machine, _ *cloud.MachineCreateDeleteData) (bool, error) {
	instance, err := p.Get(machine)
	if err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			return true, nil
		}
		return false, err
	}

	config, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return false, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	ec2Client, err := getEC2client(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return false, err
	}

	tOut, err := ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice([]string{instance.ID()}),
	})
	if err != nil {
		return false, awsErrorToTerminalError(err, "failed to terminate instance")
	}

	if *tOut.TerminatingInstances[0].PreviousState.Name != *tOut.TerminatingInstances[0].CurrentState.Name {
		glog.V(4).Infof("successfully triggered termination of instance %s at aws", instance.ID())
	}

	return false, nil
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {
	config, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	ec2Client, err := getEC2client(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return nil, err
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
		return nil, awsErrorToTerminalError(err, "failed to list instances from aws")
	}

	// We might have multiple instances (Maybe some old, terminated ones)
	// Thus we need to find the instance which is not in the terminated state
	for _, reservation := range inOut.Reservations {
		for _, i := range reservation.Instances {
			if i.State == nil || i.State.Name == nil {
				continue
			}

			if *i.State.Name == ec2.InstanceStateNameTerminated {
				continue
			}

			return &awsInstance{
				instance: i,
			}, nil
		}
	}

	return nil, cloudprovidererrors.ErrInstanceNotFound
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	c, _, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse config: %v", err)
	}

	cc := &CloudConfig{
		Global: GlobalOpts{
			VPC:      c.VpcID,
			SubnetID: c.SubnetID,
			Zone:     c.AvailabilityZone,
		},
	}

	s, err := CloudConfigToString(cc)
	if err != nil {
		return "", "", fmt.Errorf("failed to convert cloud-config to string: %v", err)
	}

	return s, "aws", nil

}

func (p *provider) MachineMetricsLabels(machine *v1alpha1.Machine) (map[string]string, error) {
	labels := make(map[string]string)

	c, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err == nil {
		labels["size"] = c.InstanceType
		labels["region"] = c.Region
		labels["az"] = c.AvailabilityZone
		labels["ami"] = c.AMI
	}

	return labels, err
}

func (p *provider) MigrateUID(machine *v1alpha1.Machine, new types.UID) error {
	instance, err := p.Get(machine)
	if err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			return nil
		}
		return fmt.Errorf("failed to get instance: %v", err)
	}

	config, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	ec2Client, err := getEC2client(config.AccessKeyID, config.SecretAccessKey, config.Region)
	if err != nil {
		return fmt.Errorf("failed to get EC2 client: %v", err)
	}

	_, err = ec2Client.CreateTags(&ec2.CreateTagsInput{
		Resources: aws.StringSlice([]string{instance.ID()}),
		Tags:      []*ec2.Tag{{Key: aws.String(machineUIDTag), Value: aws.String(string(new))}}})
	if err != nil {
		return fmt.Errorf("failed to update instance with new machineUIDTag: %v", err)
	}

	return nil
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

// awsErrorToTerminalError judges if the given error
// can be qualified as a "terminal" error, for more info see v1alpha1.MachineStatus
//
// if the given error doesn't qualify the error passed as
// an argument will be formatted according to msg and returned
func awsErrorToTerminalError(err error, msg string) error {
	prepareAndReturnError := func() error {
		return fmt.Errorf("%s, due to %s", msg, err)
	}

	if err != nil {
		aerr, ok := err.(awserr.Error)
		if !ok {
			return prepareAndReturnError()
		}
		switch aerr.Code() {
		case "InstanceLimitExceeded":
			return cloudprovidererrors.TerminalError{
				Reason:  common.InsufficientResourcesMachineError,
				Message: "You've reached the AWS quota for number of instances of this type",
			}
		case "AuthFailure":
			// authorization primitives come from MachineSpec
			// thus we are setting InvalidConfigurationMachineError
			return cloudprovidererrors.TerminalError{
				Reason:  common.InvalidConfigurationMachineError,
				Message: "A request has been rejected due to invalid credentials which were taken from the MachineSpec",
			}
		default:
			return prepareAndReturnError()
		}
	}
	return nil
}
