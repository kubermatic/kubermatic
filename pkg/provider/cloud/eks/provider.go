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

package eks

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	semverlib "github.com/Masterminds/semver/v3"
	awsprovider "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/eks/authenticator"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/client-go/tools/clientcmd/api"
)

func getAWSSession(accessKeyID, secretAccessKey, region, endpoint string) (*session.Session, error) {
	config := awsprovider.
		NewConfig().
		WithRegion(region).
		WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")).
		WithMaxRetries(3)

	// Overriding the API endpoint is mostly useful for integration tests,
	// when running against a localstack container, for example.
	if endpoint != "" {
		config = config.WithEndpoint(endpoint)
	}

	return session.NewSession(config)
}

func getClientSet(accessKeyID, secretAccessKey, region, endpoint string) (*aws.ClientSet, error) {
	sess, err := getAWSSession(accessKeyID, secretAccessKey, region, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create API session: %w", err)
	}

	return &aws.ClientSet{
		EKS: eks.New(sess),
	}, nil
}

func GetClusterConfig(ctx context.Context, accessKeyID, secretAccessKey, clusterName, region string) (*api.Config, error) {
	sess, err := getAWSSession(accessKeyID, secretAccessKey, region, "")
	if err != nil {
		return nil, err
	}
	eksSvc := eks.New(sess)

	clusterInput := &eks.DescribeClusterInput{
		Name: awsprovider.String(clusterName),
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

	gen, err := authenticator.NewGenerator(true)
	if err != nil {
		return nil, err
	}

	opts := &authenticator.GetTokenOptions{
		ClusterID: *eksclusterName,
		Session:   sess,
	}
	token, err := gen.GetWithOptions(opts)
	if err != nil {
		return nil, err
	}

	// example: eks_eu-central-1_cluster-1 => https://XX.XX.XX.XX
	name := fmt.Sprintf("eks_%s_%s", region, *eksclusterName)

	cert, err := base64.StdEncoding.DecodeString(awsprovider.StringValue(cluster.CertificateAuthority.Data))
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

func GetCredentialsForCluster(cloud kubermaticv1.ExternalClusterCloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (accessKeyID, secretAccessKey string, err error) {
	accessKeyID = cloud.EKS.AccessKeyID
	secretAccessKey = cloud.EKS.SecretAccessKey

	if accessKeyID == "" {
		if cloud.EKS.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		accessKeyID, err = secretKeySelector(cloud.EKS.CredentialsReference, resources.AWSAccessKeyID)
		if err != nil {
			return "", "", err
		}
	}

	if secretAccessKey == "" {
		if cloud.EKS.CredentialsReference == nil {
			return "", "", errors.New("no credentials provided")
		}
		secretAccessKey, err = secretKeySelector(cloud.EKS.CredentialsReference, resources.AWSSecretAccessKey)
		if err != nil {
			return "", "", err
		}
	}

	return accessKeyID, secretAccessKey, nil
}

func GetEKSClusterStatus(secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalClusterStatus, error) {
	accessKeyID, secretAccessKey, err := GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := getClientSet(accessKeyID, secretAccessKey, cloudSpec.EKS.Region, "")
	if err != nil {
		return nil, err
	}

	eksCluster, err := client.EKS.DescribeCluster(&eks.DescribeClusterInput{Name: &cloudSpec.EKS.Name})
	if err != nil {
		return nil, err
	}

	return &apiv2.ExternalClusterStatus{
		State: convertEKSStatus(*eksCluster.Cluster.Status),
	}, nil
}

func convertEKSStatus(status string) apiv2.ExternalClusterState {
	switch status {
	case "CREATING":
		return apiv2.PROVISIONING
	case "ACTIVE":
		return apiv2.RUNNING
	case "UPDATING":
		return apiv2.RECONCILING
	case "DELETING":
		return apiv2.DELETING
	case "CREATE_FAILED":
		return apiv2.ERROR
	case "DELETE_FAILED":
		return apiv2.ERROR
	case "FAILED":
		return apiv2.ERROR
	default:
		return apiv2.UNKNOWN
	}
}

func ListEKSMachineDeploymentUpgrades(ctx context.Context,
	accessKeyID, secretAccessKey, region, clusterName, machineDeployment string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)

	client, err := aws.GetClientSet(accessKeyID, secretAccessKey, "", "", region)
	if err != nil {
		return nil, err
	}
	clusterOutput, err := client.EKS.DescribeCluster(&eks.DescribeClusterInput{Name: &clusterName})
	if err != nil {
		return nil, err
	}

	if clusterOutput == nil || clusterOutput.Cluster == nil {
		return nil, fmt.Errorf("unable to get EKS cluster %s details", clusterName)
	}

	eksCluster := clusterOutput.Cluster
	if eksCluster.Version == nil {
		return nil, fmt.Errorf("unable to get EKS cluster %s version", clusterName)
	}
	currentClusterVer, err := semverlib.NewVersion(*eksCluster.Version)
	if err != nil {
		return nil, err
	}

	nodeGroupInput := &eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &machineDeployment,
	}

	nodeGroupOutput, err := client.EKS.DescribeNodegroup(nodeGroupInput)
	if err != nil {
		return nil, err
	}
	nodeGroup := nodeGroupOutput.Nodegroup

	if nodeGroup.Version == nil {
		return nil, fmt.Errorf("unable to get EKS cluster %s nodegroup %s  version", clusterName, machineDeployment)
	}
	currentMachineDeploymentVer, err := semverlib.NewVersion(*nodeGroup.Version)
	if err != nil {
		return nil, err
	}

	// return control plane version
	if currentClusterVer.GreaterThan(currentMachineDeploymentVer) {
		upgrades = append(upgrades, &apiv1.MasterVersion{Version: currentClusterVer})
	}

	return upgrades, nil
}
