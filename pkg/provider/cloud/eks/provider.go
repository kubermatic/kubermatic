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
	aws "github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/smithy-go"

	apiv1 "k8c.io/kubermatic/sdk/v2/api/v1"
	apiv2 "k8c.io/kubermatic/sdk/v2/api/v2"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/eks/authenticator"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
)

const (
	EKSNodeGroupStatus = "ACTIVE"

	ARM64Architecture = "arm64"
	X64Architecture   = "x64"

	// The architecture of the machine image.
	// Used for EKS api endpoints.
	EKSARM64Architecture  = "arm64"
	EKSX86_64Architecture = "x86_64"
)

type EKSCredentials struct {
	AccessKeyID          string
	SecretAccessKey      string
	AssumeRoleARN        string
	AssumeRoleExternalID string
}

func getClientSet(ctx context.Context, creds EKSCredentials, region, endpoint string) (*awsprovider.ClientSet, error) {
	cfg, err := awsprovider.GetAWSConfig(ctx, creds.AccessKeyID, creds.SecretAccessKey, creds.AssumeRoleARN, creds.AssumeRoleExternalID, region, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create API session: %w", err)
	}

	return &awsprovider.ClientSet{
		EKS: eks.NewFromConfig(cfg),
	}, nil
}

func GetClusterConfig(ctx context.Context, accessKeyID, secretAccessKey, clusterName, region string) (*api.Config, error) {
	cfg, err := awsprovider.GetAWSConfig(ctx, accessKeyID, secretAccessKey, "", "", region, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create API session: %w", err)
	}

	cs := awsprovider.ClientSet{EKS: eks.NewFromConfig(cfg)}

	clusterInput := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}
	clusterOutput, err := cs.EKS.DescribeCluster(ctx, clusterInput)
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
		Config:    &cfg,
	}
	token, err := gen.GetWithOptions(ctx, opts)
	if err != nil {
		return nil, err
	}

	// example: eks_eu-central-1_cluster-1 => https://XX.XX.XX.XX
	name := fmt.Sprintf("eks_%s_%s", region, *eksclusterName)

	cert, err := base64.StdEncoding.DecodeString(ptr.Deref(cluster.CertificateAuthority.Data, ""))
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
	// AWS specific configuration; use cloud platform scope.
	config.AuthInfos[name] = &api.AuthInfo{
		Token: token.Token,
	}
	return &config, nil
}

func GetCredentialsForCluster(cloud *kubermaticv1.ExternalClusterEKSCloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (EKSCredentials, error) {
	creds := EKSCredentials{
		AccessKeyID:          cloud.AccessKeyID,
		SecretAccessKey:      cloud.SecretAccessKey,
		AssumeRoleARN:        cloud.AssumeRoleARN,
		AssumeRoleExternalID: cloud.AssumeRoleExternalID,
	}
	var err error

	if creds.AccessKeyID == "" {
		if cloud.CredentialsReference == nil {
			return creds, errors.New("no credentials provided")
		}
		creds.AccessKeyID, err = secretKeySelector(cloud.CredentialsReference, resources.AWSAccessKeyID)
		if err != nil {
			return creds, nil
		}
	}

	if creds.SecretAccessKey == "" {
		if cloud.CredentialsReference == nil {
			return creds, errors.New("no credentials provided")
		}
		creds.SecretAccessKey, err = secretKeySelector(cloud.CredentialsReference, resources.AWSSecretAccessKey)
		if err != nil {
			return creds, err
		}
	}

	if creds.AssumeRoleARN == "" {
		// AssumeRoleARN is optional
		if cloud.CredentialsReference != nil {
			assumeRoleARN, err := secretKeySelector(cloud.CredentialsReference, resources.AWSAssumeRoleARN)
			if err == nil {
				creds.AssumeRoleARN = assumeRoleARN
			}
		}
	}

	if creds.AssumeRoleExternalID == "" {
		// AssumeRoleARN is optional
		if cloud.CredentialsReference != nil {
			assumeRoleExternalID, err := secretKeySelector(cloud.CredentialsReference, resources.AWSAssumeRoleExternalID)
			if err == nil {
				creds.AssumeRoleExternalID = assumeRoleExternalID
			}
		}
	}

	return creds, nil
}

func GetCluster(ctx context.Context, client *awsprovider.ClientSet, eksClusterName string) (*ekstypes.Cluster, error) {
	clusterOutput, err := client.EKS.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &eksClusterName})
	if err != nil {
		return nil, DecodeError(err)
	}
	return clusterOutput.Cluster, nil
}

func GetClusterStatus(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterEKSCloudSpec) (*kubermaticv1.ExternalClusterCondition, error) {
	creds, err := GetCredentialsForCluster(cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := getClientSet(ctx, creds, cloudSpec.Region, "")
	if err != nil {
		return nil, err
	}

	eksCluster, err := GetCluster(ctx, client, cloudSpec.Name)
	if err != nil {
		return nil, err
	}
	// check nil
	return &kubermaticv1.ExternalClusterCondition{
		Phase: ConvertStatus(eksCluster.Status),
	}, nil
}

func ListMachineDeploymentUpgrades(ctx context.Context,
	creds EKSCredentials, region, clusterName, machineDeployment string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)

	client, err := awsprovider.GetClientSet(ctx, creds.AccessKeyID, creds.SecretAccessKey, creds.AssumeRoleARN, creds.AssumeRoleExternalID, region)
	if err != nil {
		return nil, err
	}
	clusterOutput, err := client.EKS.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &clusterName})
	if err != nil {
		return nil, DecodeError(err)
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

	nodeGroupOutput, err := client.EKS.DescribeNodegroup(ctx, nodeGroupInput)
	if err != nil {
		return nil, DecodeError(err)
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

func CreateCluster(ctx context.Context, client *awsprovider.ClientSet, clusterSpec *apiv2.EKSClusterSpec, eksClusterName string) error {
	input := &eks.CreateClusterInput{
		Name: aws.String(eksClusterName),
		ResourcesVpcConfig: &ekstypes.VpcConfigRequest{
			SecurityGroupIds: clusterSpec.ResourcesVpcConfig.SecurityGroupIds,
			SubnetIds:        clusterSpec.ResourcesVpcConfig.SubnetIds,
		},
		RoleArn: aws.String(clusterSpec.RoleArn),
		Version: aws.String(clusterSpec.Version),
	}
	_, err := client.EKS.CreateCluster(ctx, input)

	if err != nil {
		return DecodeError(err)
	}
	return nil
}

func ListClusters(ctx context.Context, client *awsprovider.ClientSet) ([]string, error) {
	res, err := client.EKS.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		return nil, DecodeError(err)
	}
	return res.Clusters, nil
}

func DeleteCluster(ctx context.Context, client *awsprovider.ClientSet, eksClusterName string) error {
	_, err := client.EKS.DeleteCluster(ctx, &eks.DeleteClusterInput{Name: &eksClusterName})
	return DecodeError(err)
}

func UpgradeClusterVersion(ctx context.Context, client *awsprovider.ClientSet, version *semverlib.Version, eksClusterName string) error {
	versionString := strings.TrimSuffix(version.String(), ".0")

	updateInput := eks.UpdateClusterVersionInput{
		Name:    &eksClusterName,
		Version: &versionString,
	}
	_, err := client.EKS.UpdateClusterVersion(ctx, &updateInput)

	return DecodeError(err)
}

func CreateNodeGroup(ctx context.Context,
	client *awsprovider.ClientSet,
	clusterName, nodeGroupName string,
	eksMDCloudSpec *apiv2.EKSMachineDeploymentCloudSpec) error {
	// AmiType value is fetched from user provided AmiType value in eksMDCloudSpec
	// If user does not provides the AmiType, then AmiType is determined using the user provided machine Architecture, if provided.
	var amiType string
	switch {
	case len(eksMDCloudSpec.AmiType) > 0:
		amiType = eksMDCloudSpec.AmiType
	case eksMDCloudSpec.Architecture == EKSARM64Architecture:
		amiType = string(ekstypes.AMITypesAl2Arm64)
	case eksMDCloudSpec.Architecture == EKSX86_64Architecture:
		amiType = string(ekstypes.AMITypesAl2X8664)
	}

	createInput := &eks.CreateNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodeGroupName),
		Subnets:       eksMDCloudSpec.Subnets,
		NodeRole:      aws.String(eksMDCloudSpec.NodeRole),
		AmiType:       ekstypes.AMITypes(amiType),
		CapacityType:  ekstypes.CapacityTypes(eksMDCloudSpec.CapacityType),
		DiskSize:      aws.Int32(eksMDCloudSpec.DiskSize),
		InstanceTypes: eksMDCloudSpec.InstanceTypes,
		Labels:        eksMDCloudSpec.Labels,
		ScalingConfig: &ekstypes.NodegroupScalingConfig{
			DesiredSize: aws.Int32(eksMDCloudSpec.ScalingConfig.DesiredSize),
			MaxSize:     aws.Int32(eksMDCloudSpec.ScalingConfig.MaxSize),
			MinSize:     aws.Int32(eksMDCloudSpec.ScalingConfig.MinSize),
		},
	}
	_, err := client.EKS.CreateNodegroup(ctx, createInput)

	return DecodeError(err)
}

func ListNodegroups(ctx context.Context, client *awsprovider.ClientSet, clusterName string) ([]string, error) {
	nodeInput := &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	}
	nodeOutput, err := client.EKS.ListNodegroups(ctx, nodeInput)
	if err != nil {
		return nil, DecodeError(err)
	}
	nodeGroups := nodeOutput.Nodegroups

	return nodeGroups, nil
}

func DescribeNodeGroup(ctx context.Context, client *awsprovider.ClientSet, clusterName, nodeGroupName string) (*ekstypes.Nodegroup, error) {
	nodeGroupInput := &eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	}

	nodeGroupOutput, err := client.EKS.DescribeNodegroup(ctx, nodeGroupInput)
	if err != nil {
		return nil, DecodeError(err)
	}
	nodeGroup := nodeGroupOutput.Nodegroup

	return nodeGroup, nil
}

func UpgradeNodeGroup(ctx context.Context, client *awsprovider.ClientSet, clusterName, nodeGroupName, currentVersion, desiredVersion *string) (*eks.UpdateNodegroupVersionOutput, error) {
	nodeGroupInput := eks.UpdateNodegroupVersionInput{
		ClusterName:   clusterName,
		NodegroupName: nodeGroupName,
		Version:       desiredVersion,
	}

	updateOutput, err := client.EKS.UpdateNodegroupVersion(ctx, &nodeGroupInput)
	if err != nil {
		return nil, DecodeError(err)
	}

	return updateOutput, nil
}

func ResizeNodeGroup(ctx context.Context, client *awsprovider.ClientSet, clusterName, nodeGroupName string, currentSize, desiredSize int32) (*eks.UpdateNodegroupConfigOutput, error) {
	nodeGroup, err := DescribeNodeGroup(ctx, client, clusterName, nodeGroupName)
	if err != nil {
		return nil, err
	}
	if nodeGroup.Status != EKSNodeGroupStatus {
		return nil, fmt.Errorf("cannot resize, cluster nodegroup not active")
	}

	scalingConfig := nodeGroup.ScalingConfig
	maxSize := *scalingConfig.MaxSize
	minSize := *scalingConfig.MinSize

	var newScalingConfig ekstypes.NodegroupScalingConfig
	newScalingConfig.DesiredSize = ptr.To[int32](desiredSize)

	switch {
	case currentSize == desiredSize:
		return nil, fmt.Errorf("cluster nodes are already of size: %d", desiredSize)

	case desiredSize > maxSize:
		newScalingConfig.MaxSize = ptr.To[int32](desiredSize)

	case desiredSize < minSize:
		newScalingConfig.MinSize = ptr.To[int32](desiredSize)
	}

	configInput := eks.UpdateNodegroupConfigInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
		ScalingConfig: &newScalingConfig,
	}

	updateOutput, err := client.EKS.UpdateNodegroupConfig(ctx, &configInput)
	if err != nil {
		return nil, DecodeError(err)
	}

	return updateOutput, nil
}

func DeleteNodegroup(ctx context.Context, client *awsprovider.ClientSet, clusterName, nodeGroupName string) error {
	deleteNGInput := eks.DeleteNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	}
	_, err := client.EKS.DeleteNodegroup(ctx, &deleteNGInput)

	return DecodeError(err)
}

func ListUpgrades(ctx context.Context,
	cluster *kubermaticv1.ExternalCluster,
	clusterProvider provider.ExternalClusterProvider,
	configGetter provider.KubermaticConfigurationGetter) ([]*apiv1.MasterVersion, error) {
	upgradeVersions := []*apiv1.MasterVersion{}
	currentVersion, err := clusterProvider.GetVersion(ctx, cluster)
	if err != nil {
		return nil, err
	}
	masterVersions, err := clusterProvider.VersionsEndpoint(ctx, configGetter, kubermaticv1.EKSProviderType)
	if err != nil {
		return nil, err
	}
	for _, masterVersion := range masterVersions {
		version := masterVersion.Version
		if version.GreaterThan(currentVersion.Semver()) {
			upgradeVersions = append(upgradeVersions, &apiv1.MasterVersion{
				Version: version,
			})
		}
	}
	return upgradeVersions, nil
}

func ConvertStatus(status ekstypes.ClusterStatus) kubermaticv1.ExternalClusterPhase {
	switch status {
	case ekstypes.ClusterStatusCreating:
		return kubermaticv1.ExternalClusterPhaseProvisioning
	case ekstypes.ClusterStatusPending:
		return kubermaticv1.ExternalClusterPhaseProvisioning
	case ekstypes.ClusterStatusActive:
		return kubermaticv1.ExternalClusterPhaseRunning
	case ekstypes.ClusterStatusUpdating:
		return kubermaticv1.ExternalClusterPhaseReconciling
	case ekstypes.ClusterStatusDeleting:
		return kubermaticv1.ExternalClusterPhaseDeleting
	case ekstypes.ClusterStatusFailed:
		return kubermaticv1.ExternalClusterPhaseError
	default:
		return kubermaticv1.ExternalClusterPhaseUnknown
	}
}

func ConvertMDStatus(status ekstypes.NodegroupStatus) apiv2.ExternalClusterMDState {
	switch status {
	case ekstypes.NodegroupStatusCreating:
		return apiv2.ProvisioningExternalClusterMDState
	case ekstypes.NodegroupStatusActive:
		return apiv2.RunningExternalClusterMDState
	case ekstypes.NodegroupStatusUpdating:
		return apiv2.ReconcilingExternalClusterMDState
	case ekstypes.NodegroupStatusDeleting:
		return apiv2.DeletingExternalClusterMDState
	case ekstypes.NodegroupStatusCreateFailed:
		return apiv2.ErrorExternalClusterMDState
	case ekstypes.NodegroupStatusDeleteFailed:
		return apiv2.ErrorExternalClusterMDState
	case ekstypes.NodegroupStatusDegraded:
		return apiv2.ErrorExternalClusterMDState
	default:
		return apiv2.UnknownExternalClusterMDState
	}
}

func ValidateCredentials(ctx context.Context, credential resources.EKSCredential) error {
	client, err := awsprovider.GetClientSet(ctx, credential.AccessKeyID, credential.SecretAccessKey, "", "", credential.Region)
	if err != nil {
		return err
	}
	_, err = ListClusters(ctx, client)

	return DecodeError(err)
}

func DecodeError(err error) error {
	// Generic AWS Error with Code, Message, and original error (if any).

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		// attempt to parse error as HTTP response error to extract status code
		var httpResponseErr *awshttp.ResponseError
		if errors.As(err, &httpResponseErr) {
			return utilerrors.New(httpResponseErr.HTTPStatusCode(), fmt.Sprintf("%s: %s", apiErr.ErrorCode(), apiErr.ErrorMessage()))
		}

		// fall back to returning API error information with HTTP 500
		return utilerrors.New(500, fmt.Sprintf("%s: %s", apiErr.ErrorCode(), apiErr.ErrorMessage()))
	}

	// we were unable to parse the error as API error, returning the raw error as last resort
	return err
}
