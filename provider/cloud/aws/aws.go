package aws

import (
	"errors"
	"strconv"

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

func (a *aws) PrepareCloudSpec(*api.Cluster) error {
	panic("not implemented")
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

func (a *aws) CreateNodes(context.Context, *api.Cluster, *api.NodeSpec, int) ([]*api.Node, error) {
	panic("not implemented")
}


func (a *aws) Nodes(context.Context, *api.Cluster) ([]*api.Node, error) {
	panic("not implemented")
}


func (a *aws) DeleteNodes(ctx context.Context, c *api.Cluster, UIDs []string) error {
	panic("not implemented")
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
		if *attribute.AttributeName == "default-vpc" {
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
