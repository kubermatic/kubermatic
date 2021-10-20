/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

type ClientSet struct {
	EC2 ec2iface.EC2API
	EKS eksiface.EKSAPI
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
		EKS: eks.New(sess),
		IAM: iam.New(sess),
	}, nil
}
