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
	"strings"

	semverlib "github.com/Masterminds/semver/v3"
	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/eks/authenticator"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/client-go/tools/clientcmd/api"
)

const EKSNodeGroupStatus = "ACTIVE"

func getAWSSession(accessKeyID, secretAccessKey, region, endpoint string) (*session.Session, error) {
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

	return session.NewSession(config)
}

func getClientSet(accessKeyID, secretAccessKey, region, endpoint string) (*awsprovider.ClientSet, error) {
	sess, err := getAWSSession(accessKeyID, secretAccessKey, region, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create API session: %w", err)
	}

	return &awsprovider.ClientSet{
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

func GetClusterStatus(secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalClusterStatus, error) {
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
		return nil, DecodeAWSError(err)
	}

	return &apiv2.ExternalClusterStatus{
		State: ConvertStatus(*eksCluster.Cluster.Status),
	}, nil
}

func ConvertStatus(status string) apiv2.ExternalClusterState {
	switch status {
	case string(resources.CreatingEKSState):
		return apiv2.ProvisioningExternalClusterState
	case string(resources.PendingEKSState):
		return apiv2.ProvisioningExternalClusterState
	case string(resources.ActiveEKSState):
		return apiv2.RunningExternalClusterState
	case string(resources.UpdatingEKSState):
		return apiv2.ReconcilingExternalClusterState
	case string(resources.DeletingEKSState):
		return apiv2.DeletingExternalClusterState
	case string(resources.FailedEKSState):
		return apiv2.ErrorExternalClusterState
	default:
		return apiv2.UnknownExternalClusterState
	}
}

func ConvertMDStatus(status string) apiv2.ExternalClusterMDState {
	switch status {
	case string(resources.CreatingEKSMDState):
		return apiv2.ProvisioningExternalClusterMDState
	case string(resources.ActiveEKSMDState):
		return apiv2.RunningExternalClusterMDState
	case string(resources.UpdatingEKSMDState):
		return apiv2.ReconcilingExternalClusterMDState
	case string(resources.DeletingEKSMDState):
		return apiv2.DeletingExternalClusterMDState
	case string(resources.CreateFailedEKSMDState):
		return apiv2.ErrorExternalClusterMDState
	case string(resources.DeleteFailedEKSMDState):
		return apiv2.ErrorExternalClusterMDState
	case string(resources.DegradedEKSMDState):
		return apiv2.ErrorExternalClusterMDState
	default:
		return apiv2.UnknownExternalClusterMDState
	}
}

func ListMachineDeploymentUpgrades(ctx context.Context,
	accessKeyID, secretAccessKey, region, clusterName, machineDeployment string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", region)
	if err != nil {
		return nil, err
	}
	clusterOutput, err := client.EKS.DescribeCluster(&eks.DescribeClusterInput{Name: &clusterName})
	if err != nil {
		return nil, DecodeAWSError(err)
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
		return nil, DecodeAWSError(err)
	}
	nodeGroup := nodeGroupOutput.Nodegroup

	if nodeGroup.Version == nil {
		return nil, fmt.Errorf("unable to get EKS cluster %s nodegroup %s version", clusterName, machineDeployment)
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

func GetCluster(client *awsprovider.ClientSet, eksClusterName string) (*eks.DescribeClusterOutput, error) {
	clusterOutput, err := client.EKS.DescribeCluster(&eks.DescribeClusterInput{Name: &eksClusterName})
	if err != nil {
		return nil, DecodeAWSError(err)
	}
	return clusterOutput, nil
}

func CreateCluster(client *awsprovider.ClientSet, clusterSpec *apiv2.EKSClusterSpec, eksClusterName string) error {
	input := &eks.CreateClusterInput{
		Name: aws.String(eksClusterName),
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SecurityGroupIds: clusterSpec.ResourcesVpcConfig.SecurityGroupIds,
			SubnetIds:        clusterSpec.ResourcesVpcConfig.SubnetIds,
		},
		RoleArn: aws.String(clusterSpec.RoleArn),
		Version: aws.String(clusterSpec.Version),
	}
	_, err := client.EKS.CreateCluster(input)

	if err != nil {
		return DecodeAWSError(err)
	}
	return nil
}

func ListClusters(client *awsprovider.ClientSet) ([]*string, error) {
	req, res := client.EKS.ListClustersRequest(&eks.ListClustersInput{})
	err := req.Send()
	if err != nil {
		return nil, DecodeAWSError(err)
	}
	return res.Clusters, nil
}

func DeleteCluster(client *awsprovider.ClientSet, eksClusterName string) error {
	_, err := client.EKS.DeleteCluster(&eks.DeleteClusterInput{Name: &eksClusterName})
	return DecodeAWSError(err)
}

func UpgradeClusterVersion(client *awsprovider.ClientSet, version *semverlib.Version, eksClusterName string) error {
	versionString := strings.TrimSuffix(version.String(), ".0")

	updateInput := eks.UpdateClusterVersionInput{
		Name:    &eksClusterName,
		Version: &versionString,
	}
	_, err := client.EKS.UpdateClusterVersion(&updateInput)

	return DecodeAWSError(err)
}

func CreateNodeGroup(client *awsprovider.ClientSet,
	clusterName, nodeGroupName string,
	eksMDCloudSpec *apiv2.EKSMachineDeploymentCloudSpec) error {
	createInput := &eks.CreateNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodeGroupName),
		Subnets:       eksMDCloudSpec.Subnets,
		NodeRole:      aws.String(eksMDCloudSpec.NodeRole),
		AmiType:       aws.String(eksMDCloudSpec.AmiType),
		CapacityType:  aws.String(eksMDCloudSpec.CapacityType),
		DiskSize:      aws.Int64(eksMDCloudSpec.DiskSize),
		InstanceTypes: eksMDCloudSpec.InstanceTypes,
		Labels:        eksMDCloudSpec.Labels,
		ScalingConfig: &eks.NodegroupScalingConfig{
			DesiredSize: aws.Int64(eksMDCloudSpec.ScalingConfig.DesiredSize),
			MaxSize:     aws.Int64(eksMDCloudSpec.ScalingConfig.MaxSize),
			MinSize:     aws.Int64(eksMDCloudSpec.ScalingConfig.MinSize),
		},
	}
	_, err := client.EKS.CreateNodegroup(createInput)
	return DecodeAWSError(err)
}

func ListNodegroups(client *awsprovider.ClientSet, clusterName string) ([]*string, error) {
	nodeInput := &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	}
	nodeOutput, err := client.EKS.ListNodegroups(nodeInput)
	if err != nil {
		return nil, DecodeAWSError(err)
	}
	nodeGroups := nodeOutput.Nodegroups

	return nodeGroups, nil
}

func DescribeNodeGroup(client *awsprovider.ClientSet, clusterName, nodeGroupName string) (*eks.Nodegroup, error) {
	nodeGroupInput := &eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	}

	nodeGroupOutput, err := client.EKS.DescribeNodegroup(nodeGroupInput)
	if err != nil {
		return nil, DecodeAWSError(err)
	}
	nodeGroup := nodeGroupOutput.Nodegroup

	return nodeGroup, nil
}

func UpgradeNodeGroup(client *awsprovider.ClientSet, clusterName, nodeGroupName, currentVersion, desiredVersion *string) (*eks.UpdateNodegroupVersionOutput, error) {
	nodeGroupInput := eks.UpdateNodegroupVersionInput{
		ClusterName:   clusterName,
		NodegroupName: nodeGroupName,
		Version:       desiredVersion,
	}

	updateOutput, err := client.EKS.UpdateNodegroupVersion(&nodeGroupInput)
	if err != nil {
		return nil, DecodeAWSError(err)
	}

	return updateOutput, nil
}

func ResizeNodeGroup(client *awsprovider.ClientSet, clusterName, nodeGroupName string, currentSize, desiredSize int64) (*eks.UpdateNodegroupConfigOutput, error) {
	nodeGroup, err := DescribeNodeGroup(client, clusterName, nodeGroupName)
	if err != nil {
		return nil, err
	}
	if *nodeGroup.Status != EKSNodeGroupStatus {
		return nil, fmt.Errorf("cannot resize, cluster nodegroup not active")
	}

	scalingConfig := nodeGroup.ScalingConfig
	maxSize := *scalingConfig.MaxSize
	minSize := *scalingConfig.MinSize

	var newScalingConfig eks.NodegroupScalingConfig
	newScalingConfig.DesiredSize = &desiredSize

	switch {
	case currentSize == desiredSize:
		return nil, fmt.Errorf("cluster nodes are already of size: %d", desiredSize)

	case desiredSize > maxSize:
		newScalingConfig.MaxSize = &desiredSize

	case desiredSize < minSize:
		newScalingConfig.MinSize = &desiredSize
	}

	configInput := eks.UpdateNodegroupConfigInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
		ScalingConfig: &newScalingConfig,
	}

	updateOutput, err := client.EKS.UpdateNodegroupConfig(&configInput)
	if err != nil {
		return nil, DecodeAWSError(err)
	}

	return updateOutput, nil
}

func DeleteNodegroup(client *awsprovider.ClientSet, clusterName, nodeGroupName string) error {
	deleteNGInput := eks.DeleteNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	}
	_, err := client.EKS.DeleteNodegroup(&deleteNGInput)

	return DecodeAWSError(err)
}

func DecodeAWSError(err error) error {
	if err == nil {
		return nil
	}

	var aerr awserr.Error
	if errors.As(err, &aerr) {
		return errors.New(aerr.Message())
	}

	return err
}
