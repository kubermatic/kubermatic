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

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// The functions in this file are used throughout KKP, mostly in our REST API.

// GetSubnets returns the list of subnets for a selected AWS VPC.
func GetSubnets(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, vpcID string) ([]*ec2.Subnet, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region)
	if err != nil {
		return nil, err
	}

	subnetsInput := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{ec2VPCFilter(vpcID)},
	}

	out, err := client.EC2.DescribeSubnetsWithContext(ctx, subnetsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %w", err)
	}

	return out.Subnets, nil
}

func isAuthFailure(err error) (bool, string) {
	var awsErr awserr.Error

	if errors.As(err, &awsErr) && awsErr.Code() == authFailure {
		return true, awsErr.Message()
	}

	return false, ""
}

// GetVPCS returns the list of AWS VPCs.
func GetVPCS(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region string) ([]*ec2.Vpc, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region)
	if err != nil {
		return nil, err
	}

	vpcOut, err := client.EC2.DescribeVpcsWithContext(ctx, &ec2.DescribeVpcsInput{})

	if err != nil {
		if ok, msg := isAuthFailure(err); ok {
			return nil, utilerrors.New(401, fmt.Sprintf("failed to list VPCs: %s", msg))
		}

		return nil, fmt.Errorf("failed to list VPCs: %w", err)
	}

	return vpcOut.Vpcs, nil
}

// GetSecurityGroups returns the list of AWS Security Group filtered by VPC.
func GetSecurityGroupsByVPC(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, vpcID string) ([]*ec2.SecurityGroup, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, "", "", region)
	if err != nil {
		return nil, err
	}
	sgOut, err := client.EC2.DescribeSecurityGroupsWithContext(ctx, &ec2.DescribeSecurityGroupsInput{Filters: []*ec2.Filter{ec2VPCFilter(vpcID)}})

	if err != nil {
		if ok, msg := isAuthFailure(err); ok {
			return nil, utilerrors.New(401, fmt.Sprintf("failed to list security groups: %s", msg))
		}

		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}

	return sgOut.SecurityGroups, nil
}

// GetSecurityGroups returns the list of AWS Security Group.
func GetSecurityGroups(ctx context.Context, accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region, vpc string) ([]*ec2.SecurityGroup, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region)
	if err != nil {
		return nil, err
	}

	securityGroups, err := getSecurityGroupsWithClient(ctx, client.EC2)
	if err != nil {
		return nil, err
	}

	if vpc != "" {
		vpcSecurityGroups := make([]*ec2.SecurityGroup, 0)
		for _, sg := range securityGroups {
			if *sg.VpcId == vpc {
				vpcSecurityGroups = append(vpcSecurityGroups, sg)
			}
		}
		return vpcSecurityGroups, nil
	}

	return securityGroups, nil
}

func getSecurityGroupsWithClient(ctx context.Context, client ec2iface.EC2API) ([]*ec2.SecurityGroup, error) {
	sgOut, err := client.DescribeSecurityGroupsWithContext(ctx, &ec2.DescribeSecurityGroupsInput{})

	if err != nil {
		if ok, msg := isAuthFailure(err); ok {
			return nil, utilerrors.New(401, fmt.Sprintf("failed to list security groups: %s", msg))
		}

		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}

	return sgOut.SecurityGroups, nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID string, err error) {
	accessKeyID = cloud.AWS.AccessKeyID
	secretAccessKey = cloud.AWS.SecretAccessKey
	assumeRoleARN = cloud.AWS.AssumeRoleARN
	assumeRoleExternalID = cloud.AWS.AssumeRoleExternalID

	if accessKeyID == "" {
		if cloud.AWS.CredentialsReference == nil {
			return "", "", "", "", errors.New("no credentials provided")
		}
		accessKeyID, err = secretKeySelector(cloud.AWS.CredentialsReference, resources.AWSAccessKeyID)
		if err != nil {
			return "", "", "", "", err
		}
	}

	if secretAccessKey == "" {
		if cloud.AWS.CredentialsReference == nil {
			return "", "", "", "", errors.New("no credentials provided")
		}
		secretAccessKey, err = secretKeySelector(cloud.AWS.CredentialsReference, resources.AWSSecretAccessKey)
		if err != nil {
			return "", "", "", "", err
		}
	}

	return accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, nil
}

// GetInstanceTypes returns the list of instancetypes.
func GetInstanceTypes(accessKeyID, secretAccessKey, assumeRoleARN, assumeRoleExternalID, region string) ([]*ec2.InstanceTypeOffering, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, "", "", region)
	if err != nil {
		return nil, err
	}

	out, err := client.EC2.DescribeInstanceTypeOfferings(&ec2.DescribeInstanceTypeOfferingsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list instance types: %w", err)
	}

	return out.InstanceTypeOfferings, nil
}
