/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sts"

	"k8s.io/apimachinery/pkg/util/sets"
)

type ClientSet struct {
	EC2 ec2iface.EC2API
	EKS eksiface.EKSAPI
	IAM iamiface.IAMAPI
}

func GetClientSet(accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region string) (*ClientSet, error) {
	return getClientSet(accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, "")
}

func getAWSSession(accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, endpoint string) (*session.Session, error) {
	config := aws.
		NewConfig().
		WithRegion(region).
		WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")).
		WithMaxRetries(3)

	// Overriding the API endpoint is mostly useful for integration tests,
	// when running against a localstack container, for example.
	if endpoint != "" {
		config = config.WithEndpoint(endpoint)
	}

	awsSession, err := session.NewSession(config)
	if err != nil {
		return awsSession, err
	}

	// Assume IAM role of e.g. external AWS account if configured
	if assumeRoleARN != "" {
		return getAssumeRoleSession(awsSession, assumeRoleARN, assumeRoleExternalID, region, endpoint)
	}

	return awsSession, nil
}

func getClientSet(accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, endpoint string) (*ClientSet, error) {
	sess, err := getAWSSession(accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create API session: %w", err)
	}

	return &ClientSet{
		EC2: ec2.New(sess),
		EKS: eks.New(sess),
		IAM: iam.New(sess),
	}, nil
}

var notFoundErrors = sets.NewString("NoSuchEntity", "InvalidVpcID.NotFound", "InvalidRouteTableID.NotFound", "InvalidGroup.NotFound")

func isNotFound(err error) bool {
	var awsErr awserr.Error

	return errors.As(err, &awsErr) && notFoundErrors.Has(awsErr.Code())
}

// getAssumeRoleSession uses an existing AWS session to assume an IAM role which may be in an external AWS account.
func getAssumeRoleSession(awsSession *session.Session, assumeRoleARN, assumeRoleExternalID, region, endpoint string) (*session.Session, error) {
	assumeRoleOutput, err := getAssumeRoleCredentials(awsSession, assumeRoleARN, assumeRoleExternalID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve temporary AWS credentials for assumed role: %w", err)
	}

	assumedRoleConfig := aws.NewConfig()
	assumedRoleConfig = assumedRoleConfig.WithRegion(region)
	assumedRoleConfig = assumedRoleConfig.WithCredentials(credentials.NewStaticCredentials(*assumeRoleOutput.Credentials.AccessKeyId,
		*assumeRoleOutput.Credentials.SecretAccessKey,
		*assumeRoleOutput.Credentials.SessionToken))
	assumedRoleConfig = assumedRoleConfig.WithMaxRetries(3)

	if endpoint != "" {
		assumedRoleConfig = assumedRoleConfig.WithEndpoint(endpoint)
	}

	awsSession, err = session.NewSession(assumedRoleConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create API session with temporary credentials for an assumed IAM role: %w", err)
	}

	return awsSession, err
}

// getAssumeRoleCredentials calls the AWS Security Token Service to retrieve temporary credentials for an assumed IAM role.
func getAssumeRoleCredentials(session *session.Session, assumeRoleARN string, assumeRoleExternalID string) (*sts.AssumeRoleOutput, error) {
	stsSession := sts.New(session)
	sessionName := "kubermatic-machine-controller-assume-role"

	assumeRoleInput := sts.AssumeRoleInput{
		RoleArn:         &assumeRoleARN,
		RoleSessionName: &sessionName,
	}

	// External IDs are optional
	if assumeRoleExternalID != "" {
		assumeRoleInput.ExternalId = &assumeRoleExternalID
	}

	output, err := stsSession.AssumeRole(&assumeRoleInput)
	if err != nil {
		return nil, fmt.Errorf("failed to call AWS STS to assume IAM role: %w", err)
	}

	return output, nil
}
