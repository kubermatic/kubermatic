package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

type ClientSet struct {
	EC2 ec2iface.EC2API
	IAM iamiface.IAMAPI
}

func GetClientSet(accessKeyID, secretAccessKey, region string) (*ClientSet, error) {
	config := aws.NewConfig()
	config = config.WithRegion(region)
	config = config.WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""))
	config = config.WithMaxRetries(3)

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API session: %v", err)
	}

	return &ClientSet{
		EC2: ec2.New(sess),
		IAM: iam.New(sess),
	}, nil
}
