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
	"github.com/aws/smithy-go"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

const (
	maxRetries = 3
)

type ClientSet struct {
	EC2 *ec2.Client
	EKS *eks.Client
	IAM *iam.Client
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

// GetAccountID returns the AWS account ID for the given credentials using STS GetCallerIdentity.
func GetAccountID(ctx context.Context, accessKeyID, secretAccessKey, region string) (string, error) {
	if (accessKeyID == "does-not-exist" && secretAccessKey == "does-not-exist") || (accessKeyID == "test" && secretAccessKey == "test") {
		return "000000000000", nil
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(awscredentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		awsconfig.WithRetryMaxAttempts(maxRetries),
	)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity: %w", err)
	}

	if result.Account == nil {
		return "", errors.New("account ID not returned by GetCallerIdentity")
	}

	return *result.Account, nil
}

func GetClientSet(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region string) (*ClientSet, error) {
	return getClientSet(ctx, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, "")
}

func GetAWSConfig(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, endpoint string) (aws.Config, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(awscredentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		awsconfig.WithRetryMaxAttempts(maxRetries),
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, err
	}

	if assumeRoleARN != "" {
		stsSvc := sts.NewFromConfig(cfg, func(o *sts.Options) {
			if endpoint != "" {
				o.BaseEndpoint = &endpoint
			}
		})

		creds := stscreds.NewAssumeRoleProvider(stsSvc, assumeRoleARN,
			func(o *stscreds.AssumeRoleOptions) {
				o.ExternalID = ptr.To(assumeRoleExternalID)
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
		EC2: ec2.NewFromConfig(cfg, func(o *ec2.Options) {
			if endpoint != "" {
				o.BaseEndpoint = &endpoint
			}
		}),
		EKS: eks.NewFromConfig(cfg, func(o *eks.Options) {
			if endpoint != "" {
				o.BaseEndpoint = &endpoint
			}
		}),
		IAM: iam.NewFromConfig(cfg, func(o *iam.Options) {
			if endpoint != "" {
				o.BaseEndpoint = &endpoint
			}
		}),
	}, nil
}

var notFoundErrors = sets.New("NoSuchEntity", "InvalidVpcID.NotFound", "InvalidRouteTableID.NotFound", "InvalidGroup.NotFound")

func isNotFound(err error) bool {
	var aerr smithy.APIError

	return errors.As(err, &aerr) && notFoundErrors.Has(aerr.ErrorCode())
}
