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
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awsv1 "github.com/aws/aws-sdk-go/aws"
	awscredentialsv1 "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/smithy-go"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

const (
	maxRetries = 3
)

type ClientSet struct {
	EC2 *ec2.Client
	EKS *eks.Client
	IAM *iam.Client
}

type endpointResolver struct {
	Url string
}

func (e *endpointResolver) ResolveEndpoint(service, region string, options ...interface{}) (aws.Endpoint, error) {
	return aws.Endpoint{
		URL:           e.Url,
		SigningName:   service,
		SigningRegion: region,
	}, nil
}

func ValidateCredentials(ctx context.Context, accessKeyID, secretAccessKey string) error {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-2"),
		awsconfig.WithCredentialsProvider(awscredentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		awsconfig.WithRetryMaxAttempts(maxRetries),
	)

	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(cfg)
	_, err = client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	return err
}

func GetClientSet(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region string) (*ClientSet, error) {
	return getClientSet(ctx, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, "")
}

func GetEKSConfig(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, endpoint string) (*session.Session, error) {
	config := awsv1.
		NewConfig().
		WithRegion(region).
		WithCredentials(awscredentialsv1.NewStaticCredentials(accessKeyID, secretAccessKey, "")).
		WithMaxRetries(maxRetries)

	if endpoint != "" {
		config = config.WithEndpoint(endpoint)
	}

	return session.NewSession(config)
}

func GetAWSConfig(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, endpoint string) (aws.Config, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(awscredentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		awsconfig.WithRetryMaxAttempts(maxRetries),
	)

	if err != nil {
		return aws.Config{}, err
	}

	if endpoint != "" {
		resolver := endpointResolver{Url: endpoint}
		cfg.EndpointResolverWithOptions = &resolver
	}

	if assumeRoleARN != "" {
		stsSvc := sts.NewFromConfig(cfg)
		creds := stscreds.NewAssumeRoleProvider(stsSvc, assumeRoleARN,
			func(o *stscreds.AssumeRoleOptions) {
				o.ExternalID = pointer.String(assumeRoleExternalID)
			},
		)

		cfg.Credentials = creds
	}

	return cfg, nil
}

func getClientSet(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, endpoint string) (*ClientSet, error) {
	cfg, err := GetAWSConfig(ctx, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create API session: %w", err)
	}

	return &ClientSet{
		EC2: ec2.NewFromConfig(cfg),
		EKS: eks.NewFromConfig(cfg),
		IAM: iam.NewFromConfig(cfg),
	}, nil
}

var notFoundErrors = sets.NewString("NoSuchEntity", "InvalidVpcID.NotFound", "InvalidRouteTableID.NotFound", "InvalidGroup.NotFound")

func isNotFound(err error) bool {
	var aerr smithy.APIError

	return errors.As(err, &aerr) && notFoundErrors.Has(aerr.ErrorCode())
}
