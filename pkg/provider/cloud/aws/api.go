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
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/eks"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	httperror "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

// The functions in this file are used throughout KKP, mostly in our REST API.

// GetSubnets returns the list of subnets for a selected AWS VPC.
func GetSubnets(accessKeyID, secretAccessKey, region, vpcID string) ([]*ec2.Subnet, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	subnetsInput := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{ec2VPCFilter(vpcID)},
	}

	out, err := client.EC2.DescribeSubnets(subnetsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %w", err)
	}

	return out.Subnets, nil
}

// GetVPCS returns the list of AWS VPCs.
func GetVPCS(accessKeyID, secretAccessKey, region string) ([]*ec2.Vpc, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	vpcOut, err := client.EC2.DescribeVpcs(&ec2.DescribeVpcsInput{})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == authFailure {
			return nil, httperror.New(401, fmt.Sprintf("failed to list VPCs: %s", awsErr.Message()))
		}

		return nil, fmt.Errorf("failed to list VPCs: %w", err)
	}

	return vpcOut.Vpcs, nil
}

// GetSecurityGroups returns the list of AWS Security Group.
func GetSecurityGroups(accessKeyID, secretAccessKey, region string) ([]*ec2.SecurityGroup, error) {
	client, err := GetClientSet(accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	return getSecurityGroupsWithClient(client.EC2)
}

func getSecurityGroupsWithClient(client ec2iface.EC2API) ([]*ec2.SecurityGroup, error) {
	sgOut, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == authFailure {
			return nil, httperror.New(401, fmt.Sprintf("failed to list security groups: %s", awsErr.Message()))
		}

		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}

	return sgOut.SecurityGroups, nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (accessKeyID, secretAccessKey string, err error) {
	accessKeyID = cloud.AWS.AccessKeyID
	secretAccessKey = cloud.AWS.SecretAccessKey

	if accessKeyID == "" {
		if cloud.AWS.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		accessKeyID, err = secretKeySelector(cloud.AWS.CredentialsReference, resources.AWSAccessKeyID)
		if err != nil {
			return "", "", err
		}
	}

	if secretAccessKey == "" {
		if cloud.AWS.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		secretAccessKey, err = secretKeySelector(cloud.AWS.CredentialsReference, resources.AWSSecretAccessKey)
		if err != nil {
			return "", "", err
		}
	}

	return accessKeyID, secretAccessKey, nil
}

func GetEKSClusterConfig(ctx context.Context, accessKeyID, secretAccessKey, clusterName, region string) (*api.Config, error) {
	sess, err := getAWSSession(accessKeyID, secretAccessKey, region, "")
	if err != nil {
		return nil, err
	}
	eksSvc := eks.New(sess)

	clusterInput := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}
	clusterOutput, err := eksSvc.DescribeCluster(clusterInput)
	if err != nil {
		return nil, fmt.Errorf("error calling DescribeCluster: %w", err)
	}

	cluster := clusterOutput.Cluster
	eksclusterName := cluster.Name

	config := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   map[string]*api.Cluster{},
		AuthInfos:  map[string]*api.AuthInfo{},
		Contexts:   map[string]*api.Context{},
	}

	gen, err := token.NewGenerator(true, false)
	if err != nil {
		return nil, err
	}

	opts := &token.GetTokenOptions{
		ClusterID: *eksclusterName,
		Session:   sess,
	}
	token, err := gen.GetWithOptions(opts)
	if err != nil {
		return nil, err
	}

	// example: eks_eu-central-1_cluster-1 => https://XX.XX.XX.XX
	name := fmt.Sprintf("eks_%s_%s", region, *eksclusterName)

	cert, err := base64.StdEncoding.DecodeString(aws.StringValue(cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, err
	}

	config.Clusters[name] = &api.Cluster{
		CertificateAuthorityData: cert,
		Server:                   *cluster.Endpoint,
	}
	config.CurrentContext = name

	// Just reuse the context name as an auth name.
	config.Contexts[name] = &api.Context{
		Cluster:  name,
		AuthInfo: name,
	}
	// AWS specific configation; use cloud platform scope.
	config.AuthInfos[name] = &api.AuthInfo{
		Token: token.Token,
	}
	return &config, nil
}
