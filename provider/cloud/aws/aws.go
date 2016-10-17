package aws

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"text/template"

	ktemplate "github.com/kubermatic/api/template"
	"golang.org/x/net/context"

	sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
)

const (
	accessKeyIDAnnotationKey     = "acccess-key-id"
	secretAccessKeyAnnotationKey = "secret-access-key"
	sshAnnotationKey             = "ssh-key"
	awsKeyDelimitor              = ","
)

const (
	awsFilterName       = "Name"
	awsFilterState      = "instance-state-name"
	awsFilterRunning    = "running"
	awsFilterPending    = "pending"
	awsFilterDefaultVPC = "default-vpc"
)

const (
	// TODO: Create aws image
	awsLoodseImageName = ""

	defaultDeviceName   = "/dev/sda1"
	defaultInstanceType = "t2.micro"
)

var defaultCreatorTagLoodse = &ec2.Tag{
	Key:   sdk.String("controller"),
	Value: sdk.String("kubermatic"),
}

type aws struct {
	datacenters map[string]provider.DatacenterMeta
}

// NewCloudProvider returns a new aws provider.
func NewCloudProvider(datacenters map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &aws{
		datacenters: datacenters,
	}
}

func (a *aws) PrepareCloudSpec(cluster *api.Cluster) error {
	svc := getSession(cluster)
	for i, cert := range cluster.Spec.Cloud.AWS.SSHKeys {
		params := &ec2.ImportKeyPairInput{
			KeyName:           sdk.String(fmt.Sprintf("Loodse-%d", i)),
			PublicKeyMaterial: []byte(cert),
		}
		_, err := svc.ImportKeyPair(params)
		if err != nil {
			continue
		}

	}
	return nil
}

func (*aws) CreateAnnotations(cs *api.CloudSpec) (annotations map[string]string, err error) {
	return map[string]string{
		accessKeyIDAnnotationKey:     strconv.FormatInt(cs.AWS.AccessKeyID, 10),
		secretAccessKeyAnnotationKey: cs.AWS.SecretAccessKey,
	}, nil
}

func (*aws) Cloud(annotations map[string]string) (*api.CloudSpec, error) {
	spec := &api.CloudSpec{
		AWS: &api.AWSCloudSpec{},
	}
	var ok bool
	if val, ok := annotations[accessKeyIDAnnotationKey]; ok {
		spec.AWS.AccessKeyID, _ = strconv.ParseInt(val, 10, 64)
	} else {
		return nil, errors.New("no access key ID found")
	}
	if spec.AWS.SecretAccessKey, ok = annotations[secretAccessKeyAnnotationKey]; !ok {
		return nil, errors.New("no secret key found")
	}
	return spec, nil
}

func (a *aws) CreateNodes(ctx context.Context, cluster *api.Cluster, node *api.NodeSpec, num int) ([]*api.Node, error) {
	dc, ok := a.datacenters[node.DC]
	if !ok || dc.Spec.AWS == nil {
		return nil, nil
	}

	if node.AWS.Type != "" {
		return nil, nil
	}

	svc := getSession(cluster)
	cSpec := cluster.Spec.Cloud.GetAWS()
	nSpec := node.AWS

	var created []*api.Node

	for i := 0; i < num; i++ {
		id := provider.ShortUID(5)
		instanceName := fmt.Sprintf("kubermatic-%s-%s", cluster.Metadata.Name, id)

		clientKC, err := cluster.CreateKeyCert(instanceName)
		if err != nil {
			return created, err
		}

		block := &ec2.BlockDeviceMapping{
			DeviceName: sdk.String(defaultDeviceName),
			Ebs: &ec2.EbsBlockDevice{
				DeleteOnTermination: sdk.Bool(true),
				VolumeSize:          sdk.Int64(nSpec.Size),
				VolumeType:          sdk.String(defaultInstanceType),
			},
		}
		data := ktemplate.Data{
			DC:                node.DC,
			ClusterName:       cluster.Metadata.Name,
			SSHAuthorizedKeys: cSpec.SSHKeys,
			EtcdURL:           cluster.Address.EtcdURL,
			APIServerURL:      cluster.Address.URL,
			Region:            dc.Spec.AWS.AvailabilityZone,
			Name:              instanceName,
			ClientKey:         clientKC.Key.Base64(),
			ClientCert:        clientKC.Cert.Base64(),
			RootCACert:        cluster.Status.RootCA.Cert.Base64(),
			ApiserverPubSSH:   cluster.Status.ApiserverSSH,
			ApiserverToken:    cluster.Address.Token,
			FlannelCIDR:       cluster.Spec.Cloud.Network.Flannel.CIDR,
		}

		tpl, err := template.New("cloud-config-node.yaml").Funcs(ktemplate.FuncMap).ParseFiles("template/coreos/cloud-config-node.yaml")

		if err != nil {
			return created, err
		}

		var buf bytes.Buffer
		if err = tpl.Execute(&buf, data); err != nil {
			return created, err
		}

		instanceRequest := &ec2.RunInstancesInput{
			ImageId:             sdk.String(awsLoodseImageName),
			MaxCount:            sdk.Int64(1),
			MinCount:            sdk.Int64(1),
			BlockDeviceMappings: []*ec2.BlockDeviceMapping{block},
			InstanceType:        sdk.String(defaultInstanceType),
			Placement: &ec2.Placement{
				AvailabilityZone: sdk.String(node.DC),
			},
			UserData: sdk.String(buf.String()),
		}

		node, err := launch(svc, instanceName, instanceRequest)
		if err == nil {
			created = append(created, node)
		}
	}
	return created, nil
}

func (a *aws) Nodes(ctx context.Context, cluster *api.Cluster) ([]*api.Node, error) {
	svc := getSession(cluster)

	var nodes []*api.Node

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

	resp, err := svc.DescribeInstances(params)
	if err != nil {
		return nil, err
	}

	for _, n := range resp.Reservations {
		for _, instance := range n.Instances {
			var isOwner bool
			var name string
			// Max 10 iterations
			for _, tag := range instance.Tags {
				if *tag.Key == *defaultCreatorTagLoodse.Key && *tag.Value == *defaultCreatorTagLoodse.Value {
					isOwner = true
				}
				if *tag.Key == awsFilterName {
					name = *tag.Value
				}
			}
			if !isOwner {
				continue
			}

			node := &api.Node{
				Metadata: api.Metadata{
					UID:  *instance.InstanceId,
					Name: name,
				},
				Status: api.NodeStatus{
					Addresses: map[string]string{
						"public":  *instance.PublicIpAddress,
						"private": *instance.PrivateIpAddress,
					},
				},
				Spec: api.NodeSpec{
					DC: *instance.Placement.AvailabilityZone,
					AWS: &api.AWSNodeSpec{
						Type: *instance.ImageId,
						// TODO: Get BlockDeviceMapping Volume Size
						Size: -1,
					},
				},
			}
			nodes = append(nodes, node)
		}
	}
	return nil, nil
}

func (a *aws) DeleteNodes(ctx context.Context, cluster *api.Cluster, UIDs []string) error {
	svc := getSession(cluster)

	awsInstanceIds := make([]*string, len(UIDs))
	for i := 0; i < len(UIDs); i++ {
		awsInstanceIds[i] = sdk.String(UIDs[i])
	}

	terminateRequest := &ec2.TerminateInstancesInput{
		InstanceIds: awsInstanceIds,
	}

	_, err := svc.TerminateInstances(terminateRequest)
	return err
}

func launch(client *ec2.EC2, name string, instance *ec2.RunInstancesInput) (*api.Node, error) {
	serverReq, err := client.RunInstances(instance)
	if err != nil {
		return nil, err
	}

	_, err = client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{serverReq.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   sdk.String(awsLoodseImageName),
				Value: sdk.String(name),
			},
			defaultCreatorTagLoodse,
		},
	})

	if err != nil {
		return nil, err
	}

	return &api.Node{
		Metadata: api.Metadata{
			// This looks weird but is correct
			UID:  *serverReq.Instances[0].InstanceId,
			Name: name,
		},
		Status: api.NodeStatus{
			Addresses: map[string]string{
				// Probably won't have one... VPC ?
				// TODO: VPC rules ... NetworkInterfaces
				"public":  *serverReq.Instances[0].PublicIpAddress,
				"private": *serverReq.Instances[0].PrivateIpAddress,
			},
		},
		Spec: api.NodeSpec{
			DC: *instance.Placement.AvailabilityZone,
			AWS: &api.AWSNodeSpec{
				Type: *instance.ImageId,
				Size: *instance.BlockDeviceMappings[0].Ebs.VolumeSize,
			},
		},
	}, nil
}

func getSession(cluster *api.Cluster) *ec2.EC2 {
	awsSpec := cluster.Spec.Cloud.GetAWS()
	config := sdk.NewConfig()
	config = config.WithRegion(cluster.Spec.Cloud.DC)
	config = config.WithCredentials(credentials.NewStaticCredentials(strconv.FormatInt(awsSpec.AccessKeyID, 10), awsSpec.SecretAccessKey, ""))
	// TODO: specify retrycount
	config = config.WithMaxRetries(3)
	return ec2.New(session.New(config))
}

func getDefaultVPCId(client *ec2.EC2) (string, error) {
	output, err := client.DescribeAccountAttributes(&ec2.DescribeAccountAttributesInput{})
	if err != nil {
		return "", err
	}

	for _, attribute := range output.AccountAttributes {
		if *attribute.AttributeName == awsFilterDefaultVPC {
			return *attribute.AttributeValues[0].AttributeValue, nil
		}
	}

	return "", errors.New("No default-vpc attribute")
}

func makePointerSlice(stackSlice []string) []*string {
	pointerSlice := []*string{}
	for i := range stackSlice {
		pointerSlice = append(pointerSlice, &stackSlice[i])
	}
	return pointerSlice
}
