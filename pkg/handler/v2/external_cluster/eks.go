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

package externalcluster

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/go-kit/kit/endpoint"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	eksprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/eks"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
)

const (
	EKSNodeGroupStatus    = "ACTIVE"
	EKSNodeGroupNameLabel = "eks.amazonaws.com/nodegroup"
	EKSAMITypes           = "Amazon Linux 2"
	EKSCustomAMIType      = "CUSTOM"
	EKSCapacityTypes      = "SPOT"
)

// EKSCommonReq represent a request with common parameters for EKS.
type EKSCommonReq struct {
	// in: header
	// name: AccessKeyID
	AccessKeyID string
	// in: header
	// name: SecretAccessKey
	SecretAccessKey string
	// in: header
	// name: Credential
	Credential string
}

func DecodeEKSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSCommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.SecretAccessKey = r.Header.Get("SecretAccessKey")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

// EKSTypesReq represent a request for EKS types.
// swagger:parameters validateEKSCredentials listEKSRegions listEKSVPCS
type EKSTypesReq struct {
	EKSCommonReq
	// in: header
	// name: Region
	Region string
}

func DecodeEKSTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSTypesReq

	commonReq, err := DecodeEKSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.Region = r.Header.Get("Region")
	req.EKSCommonReq = commonReq.(EKSCommonReq)

	return req, nil
}

// EKSClusterListReq represent a request for EKS cluster list.
// swagger:parameters listEKSClusters
type EKSClusterListReq struct {
	common.ProjectReq
	EKSTypesReq
}

func (req EKSTypesReq) Validate() error {
	if len(req.Credential) != 0 {
		return nil
	}
	if len(req.AccessKeyID) == 0 || len(req.SecretAccessKey) == 0 || len(req.Region) == 0 {
		return fmt.Errorf("EKS Credentials or Region cannot be empty")
	}
	return nil
}

func (req EKSReq) Validate() error {
	if len(req.VpcId) == 0 {
		return fmt.Errorf("EKS VPC ID cannot be empty")
	}
	if len(req.Credential) != 0 {
		return nil
	}
	if len(req.AccessKeyID) == 0 || len(req.SecretAccessKey) == 0 {
		return fmt.Errorf("EKS Credentials cannot be empty")
	}
	if len(req.Region) == 0 {
		return fmt.Errorf("Region cannot be empty")
	}
	return nil
}

func DecodeEKSClusterListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSClusterListReq

	typesReq, err := DecodeEKSTypesReq(c, r)
	if err != nil {
		return nil, err
	}
	req.EKSTypesReq = typesReq.(EKSTypesReq)
	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

// eksNoCredentialReq represent a request for EKS resources
// swagger:parameters listEKSVPCsNoCredentials listEKSInstanceTypesNoCredentials
type eksNoCredentialReq struct {
	GetClusterReq
}

func DecodeEKSNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req eksNoCredentialReq
	re, err := DecodeGetReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = re.(GetClusterReq)

	return req, nil
}

// eksSubnetsNoCredentialReq represent a request for EKS resources
// swagger:parameters listEKSSubnetsNoCredentials
type eksSubnetsNoCredentialReq struct {
	eksNoCredentialReq
	// in: header
	// name: VpcId
	VpcId string
}

func DecodeEKSSubnetsNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req eksSubnetsNoCredentialReq
	re, err := DecodeEKSNoCredentialReq(c, r)
	if err != nil {
		return nil, err
	}
	req.eksNoCredentialReq = re.(eksNoCredentialReq)
	req.VpcId = r.Header.Get("VpcId")

	return req, nil
}

// Validate validates eksSubnetsNoCredentialReq request.
func (req eksSubnetsNoCredentialReq) Validate() error {
	if err := req.GetClusterReq.Validate(); err != nil {
		return err
	}
	if len(req.VpcId) == 0 {
		return fmt.Errorf("AKS VPC ID cannot be empty")
	}
	return nil
}

func ListEKSClustersEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSClusterListReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}
		credential, err := getEKSCredentialsFromReq(ctx, req.EKSTypesReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSClusters(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, clusterProvider, *credential, req.ProjectID)
	}
}

func ListEKSVPCEndpoint(userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSTypesReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential, err := getEKSCredentialsFromReq(ctx, req, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSVPC(ctx, *credential)
	}
}

func ListEKSSubnetsEndpoint(userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential, err := getEKSCredentialsFromReq(ctx, req.EKSTypesReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSSubnetIDs(ctx, *credential, req.VpcId)
	}
}

func ListEKSSecurityGroupsEndpoint(userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential, err := getEKSCredentialsFromReq(ctx, req.EKSTypesReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSSecurityGroupIDs(ctx, *credential, req.VpcId)
	}
}

func ListEKSRegionsEndpoint(userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSTypesReq)

		credential, err := getEKSCredentialsFromReq(ctx, req, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListEKSRegions(ctx, *credential)
	}
}

func EKSValidateCredentialsEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(EKSTypesReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential, err := getEKSCredentialsFromReq(ctx, req, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return nil, providercommon.ValidateEKSCredentials(ctx, *credential)
	}
}

func getEKSCredentialsFromReq(ctx context.Context, req EKSTypesReq, userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) (*providercommon.EKSCredential, error) {
	accessKeyID := req.AccessKeyID
	secretAccessKey := req.SecretAccessKey
	region := req.Region

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	presetName := req.Credential
	if len(presetName) > 0 {
		preset, err := presetProvider.GetPreset(ctx, userInfo, presetName)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", presetName, userInfo.Email))
		}
		if credentials := preset.Spec.EKS; credentials != nil {
			accessKeyID = credentials.AccessKeyID
			secretAccessKey = credentials.SecretAccessKey
			region = credentials.Region
		}
	}

	return &providercommon.EKSCredential{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
	}, nil
}

// EKSSubnetsReq represent a request for EKS subnets.
// swagger:parameters listEKSSubnets listEKSSecurityGroups
type EKSReq struct {
	EKSTypesReq
	// in: header
	// name: VpcId
	VpcId string
}

func DecodeEKSReq(c context.Context, r *http.Request) (interface{}, error) {
	var req EKSReq

	typesReq, err := DecodeEKSTypesReq(c, r)
	if err != nil {
		return nil, err
	}
	req.EKSTypesReq = typesReq.(EKSTypesReq)
	req.VpcId = r.Header.Get("VpcId")

	return req, nil
}

func createNewEKSCluster(ctx context.Context, eksClusterSpec *apiv2.EKSClusterSpec, eksCloudSpec *apiv2.EKSCloudSpec) error {
	client, err := awsprovider.GetClientSet(eksCloudSpec.AccessKeyID, eksCloudSpec.SecretAccessKey, "", "", eksCloudSpec.Region)
	if err != nil {
		return err
	}

	clusterSpec := eksClusterSpec

	fields := reflect.ValueOf(clusterSpec).Elem()
	for i := 0; i < fields.NumField(); i++ {
		yourjsonTags := fields.Type().Field(i).Tag.Get("required")
		if strings.Contains(yourjsonTags, "true") && fields.Field(i).IsZero() {
			return fmt.Errorf("required field is missing %v", fields.Type().Field(i).Tag)
		}
	}
	input := &eks.CreateClusterInput{
		Name: aws.String(eksCloudSpec.Name),
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SecurityGroupIds: clusterSpec.ResourcesVpcConfig.SecurityGroupIds,
			SubnetIds:        clusterSpec.ResourcesVpcConfig.SubnetIds,
		},
		RoleArn: aws.String(clusterSpec.RoleArn),
		Version: aws.String(clusterSpec.Version),
	}
	_, err = client.EKS.CreateCluster(input)

	return err
}

func createOrImportEKSCluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticv1.Project, spec *apiv2.ExternalClusterSpec, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticv1.ExternalCluster, error) {
	fields := reflect.ValueOf(cloud.EKS).Elem()
	for i := 0; i < fields.NumField(); i++ {
		yourjsonTags := fields.Type().Field(i).Tag.Get("required")
		if strings.Contains(yourjsonTags, "true") && fields.Field(i).IsZero() {
			return nil, utilerrors.NewBadRequest("required field is missing: %v", fields.Type().Field(i).Name)
		}
	}

	if spec != nil && spec.EKSClusterSpec != nil {
		if err := createNewEKSCluster(ctx, spec.EKSClusterSpec, cloud.EKS); err != nil {
			return nil, err
		}
	}

	newCluster := genExternalCluster(name, project.Name)
	newCluster.Spec.CloudSpec = &kubermaticv1.ExternalClusterCloudSpec{
		EKS: &kubermaticv1.ExternalClusterEKSCloudSpec{
			Name:   cloud.EKS.Name,
			Region: cloud.EKS.Region,
		},
	}

	keyRef, err := clusterProvider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, project.Name, newCluster.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	kuberneteshelper.AddFinalizer(newCluster, apiv1.CredentialsSecretsCleanupFinalizer)
	newCluster.Spec.CloudSpec.EKS.CredentialsReference = keyRef

	return createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
}

func patchEKSCluster(oldCluster, newCluster *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalCluster, error) {
	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	newVersion := newCluster.Spec.Version.Semver()
	newVersionString := strings.TrimSuffix(newVersion.String(), ".0")

	updateInput := eks.UpdateClusterVersionInput{
		Name:    &cloudSpec.EKS.Name,
		Version: &newVersionString,
	}
	_, err = client.EKS.UpdateClusterVersion(&updateInput)
	if err != nil {
		return nil, err
	}

	return newCluster, nil
}

func getEKSNodeGroups(ctx context.Context, cluster *kubermaticv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	cloudSpec := cluster.Spec.CloudSpec
	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	clusterName := cloudSpec.EKS.Name
	nodeInput := &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	}
	nodeOutput, err := client.EKS.ListNodegroups(nodeInput)
	if err != nil {
		return nil, err
	}
	nodeGroups := nodeOutput.Nodegroups

	machineDeployments := make([]apiv2.ExternalClusterMachineDeployment, 0, len(nodeGroups))

	nodes, err := clusterProvider.ListNodes(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	for _, nodeGroupName := range nodeGroups {
		var readyReplicas int32
		for _, n := range nodes.Items {
			if n.Labels != nil {
				if n.Labels[EKSNodeGroupNameLabel] == *nodeGroupName {
					readyReplicas++
				}
			}
		}

		nodeGroupInput := &eks.DescribeNodegroupInput{
			ClusterName:   &clusterName,
			NodegroupName: nodeGroupName,
		}

		nodeGroupOutput, err := client.EKS.DescribeNodegroup(nodeGroupInput)
		if err != nil {
			return nil, err
		}
		nodeGroup := nodeGroupOutput.Nodegroup
		machineDeployments = append(machineDeployments, createMachineDeploymentFromEKSNodePoll(nodeGroup, readyReplicas))
	}

	return machineDeployments, err
}

func getEKSNodeGroup(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodeGroupName string, secretKeySelector provider.SecretKeySelectorValueFunc, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	cloudSpec := cluster.Spec.CloudSpec

	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	return getEKSMachineDeployment(ctx, client, cluster, nodeGroupName, clusterProvider)
}

func getEKSMachineDeployment(ctx context.Context, client *awsprovider.ClientSet, cluster *kubermaticv1.ExternalCluster, nodeGroupName string, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	clusterName := cluster.Spec.CloudSpec.EKS.Name

	nodeGroupInput := &eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	}

	nodeGroupOutput, err := client.EKS.DescribeNodegroup(nodeGroupInput)
	if err != nil {
		return nil, err
	}
	nodeGroup := nodeGroupOutput.Nodegroup

	nodes, err := clusterProvider.ListNodes(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var readyReplicas int32
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[EKSNodeGroupNameLabel] == nodeGroupName {
				readyReplicas++
			}
		}
	}
	machineDeployment := createMachineDeploymentFromEKSNodePoll(nodeGroup, readyReplicas)

	return &machineDeployment, err
}

func createMachineDeploymentFromEKSNodePoll(nodeGroup *eks.Nodegroup, readyReplicas int32) apiv2.ExternalClusterMachineDeployment {
	md := apiv2.ExternalClusterMachineDeployment{
		NodeDeployment: apiv1.NodeDeployment{
			ObjectMeta: apiv1.ObjectMeta{
				ID:   aws.StringValue(nodeGroup.NodegroupName),
				Name: aws.StringValue(nodeGroup.NodegroupName),
			},
			Spec: apiv1.NodeDeploymentSpec{
				Template: apiv1.NodeSpec{
					Versions: apiv1.NodeVersionInfo{
						Kubelet: aws.StringValue(nodeGroup.Version),
					},
				},
			},
			Status: clusterv1alpha1.MachineDeploymentStatus{
				ReadyReplicas: readyReplicas,
			},
		},
		Cloud: &apiv2.ExternalClusterMachineDeploymentCloudSpec{},
	}
	md.Cloud.EKS = &apiv2.EKSMachineDeploymentCloudSpec{
		Subnets:       nodeGroup.Subnets,
		NodeRole:      aws.StringValue(nodeGroup.NodeRole),
		AmiType:       aws.StringValue(nodeGroup.AmiType),
		CapacityType:  aws.StringValue(nodeGroup.CapacityType),
		DiskSize:      aws.Int64Value(nodeGroup.DiskSize),
		InstanceTypes: nodeGroup.InstanceTypes,
		Labels:        nodeGroup.Labels,
		Version:       aws.StringValue(nodeGroup.Version),
	}
	scalingConfig := nodeGroup.ScalingConfig
	if scalingConfig != nil {
		md.Spec.Replicas = int32(aws.Int64Value(scalingConfig.DesiredSize))
		md.Status.Replicas = int32(aws.Int64Value(scalingConfig.DesiredSize))
		md.Cloud.EKS.ScalingConfig = apiv2.EKSNodegroupScalingConfig{
			DesiredSize: aws.Int64Value(scalingConfig.DesiredSize),
			MaxSize:     aws.Int64Value(scalingConfig.MaxSize),
			MinSize:     aws.Int64Value(scalingConfig.MinSize),
		}
	}

	return md
}

func patchEKSMachineDeployment(oldMD, newMD *apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, cluster *kubermaticv1.ExternalCluster) (*apiv2.ExternalClusterMachineDeployment, error) {
	cloudSpec := cluster.Spec.CloudSpec

	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	// The EKS can update Node Group size or NodeGroup object. Can't change both till one of the update is in progress.
	// It's required to update NodeGroup size separately.

	clusterName := cloudSpec.EKS.Name
	nodeGroupName := newMD.NodeDeployment.Name

	currentReplicas := oldMD.NodeDeployment.Spec.Replicas
	desiredReplicas := newMD.NodeDeployment.Spec.Replicas
	currentVersion := oldMD.NodeDeployment.Spec.Template.Versions.Kubelet
	desiredVersion := newMD.NodeDeployment.Spec.Template.Versions.Kubelet
	if desiredReplicas != currentReplicas {
		_, err = resizeEKSNodeGroup(client, clusterName, nodeGroupName, int64(currentReplicas), int64(desiredReplicas))
		if err != nil {
			return nil, err
		}
		newMD.NodeDeployment.Status.Replicas = desiredReplicas
		newMD.NodeDeployment.Spec.Template.Versions.Kubelet = currentVersion
		return newMD, nil
	}

	if desiredVersion != currentVersion {
		_, err = upgradeEKSNodeGroup(client, &clusterName, &nodeGroupName, &currentVersion, &desiredVersion)
		if err != nil {
			return nil, err
		}
		newMD.NodeDeployment.Spec.Replicas = currentReplicas
		return newMD, nil
	}

	return newMD, nil
}

func upgradeEKSNodeGroup(client *awsprovider.ClientSet, clusterName, nodeGroupName, currentVersion, desiredVersion *string) (*eks.UpdateNodegroupVersionOutput, error) {
	nodeGroupInput := eks.UpdateNodegroupVersionInput{
		ClusterName:   clusterName,
		NodegroupName: nodeGroupName,
		Version:       desiredVersion,
	}

	updateOutput, err := client.EKS.UpdateNodegroupVersion(&nodeGroupInput)
	if err != nil {
		return nil, err
	}

	return updateOutput, nil
}

func resizeEKSNodeGroup(client *awsprovider.ClientSet, clusterName, nodeGroupName string, currentSize, desiredSize int64) (*eks.UpdateNodegroupConfigOutput, error) {
	nodeGroupInput := eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	}

	nodeGroupOutput, err := client.EKS.DescribeNodegroup(&nodeGroupInput)
	if err != nil {
		return nil, err
	}

	nodeGroup := nodeGroupOutput.Nodegroup
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
		return nil, err
	}

	return updateOutput, nil
}

func getEKSNodes(ctx context.Context,
	cluster *kubermaticv1.ExternalCluster,
	nodeGroupName string,
	clusterProvider provider.ExternalClusterProvider,
) ([]corev1.Node, error) {
	var outputNodes []corev1.Node

	nodes, err := clusterProvider.ListNodes(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[EKSNodeGroupNameLabel] == nodeGroupName {
				outputNodes = append(outputNodes, n)
			}
		}
	}

	return outputNodes, err
}

func deleteEKSNodeGroup(cluster *kubermaticv1.ExternalCluster, nodeGroupName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) error {
	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cluster.Spec.CloudSpec, secretKeySelector)
	if err != nil {
		return err
	}

	cloudSpec := cluster.Spec.CloudSpec
	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return err
	}

	deleteNGInput := eks.DeleteNodegroupInput{
		ClusterName:   &cloudSpec.EKS.Name,
		NodegroupName: &nodeGroupName,
	}
	_, err = client.EKS.DeleteNodegroup(&deleteNGInput)

	return err
}

func EKSInstanceTypesWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(eksNoCredentialReq)
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

		cloudSpec := cluster.Spec.CloudSpec
		if cloudSpec.EKS == nil {
			return nil, utilerrors.NewNotFound("cloud spec for %s", cluster.Name)
		}

		accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
		if err != nil {
			return nil, err
		}

		if cloudSpec.EKS.Region == "" {
			return nil, errors.New("no region provided in externalcluter spec")
		}
		credential := providercommon.EKSCredential{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			Region:          cloudSpec.EKS.Region,
		}
		return providercommon.ListInstanceTypes(ctx, credential)
	}
}

func EKSVPCsWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(eksNoCredentialReq)
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

		cloudSpec := cluster.Spec.CloudSpec
		if cloudSpec.EKS == nil {
			return nil, utilerrors.NewNotFound("cloud spec for %s", cluster.Name)
		}

		accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
		if err != nil {
			return nil, err
		}

		if cloudSpec.EKS.Region == "" {
			return nil, errors.New("no region provided in externalcluter spec")
		}
		credential := providercommon.EKSCredential{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			Region:          cloudSpec.EKS.Region,
		}
		return providercommon.ListEKSVPC(ctx, credential)
	}
}

func EKSSubnetsWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(eksSubnetsNoCredentialReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}
		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())

		cloudSpec := cluster.Spec.CloudSpec
		if cloudSpec.EKS == nil {
			return nil, utilerrors.NewNotFound("cloud spec for %s", cluster.Name)
		}

		accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
		if err != nil {
			return nil, err
		}

		if cloudSpec.EKS.Region == "" {
			return nil, errors.New("no region provided in externalcluter spec")
		}
		cred := providercommon.EKSCredential{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			Region:          cloudSpec.EKS.Region,
		}
		return providercommon.ListEKSSubnetIDs(ctx, cred, req.VpcId)
	}
}

func getEKSClusterDetails(ctx context.Context, apiCluster *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalCluster, error) {
	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}
	clusterOutput, err := client.EKS.DescribeCluster(&eks.DescribeClusterInput{Name: &cloudSpec.EKS.Name})
	if err != nil {
		return nil, err
	}
	eksCluster := clusterOutput.Cluster
	if eksCluster == nil {
		return apiCluster, nil
	}

	clusterSpec := &apiv2.EKSClusterSpec{
		RoleArn: aws.StringValue(eksCluster.RoleArn),
		Version: aws.StringValue(eksCluster.Version),
	}

	if eksCluster.ResourcesVpcConfig != nil {
		clusterSpec.ResourcesVpcConfig = apiv2.VpcConfigRequest{
			SecurityGroupIds: eksCluster.ResourcesVpcConfig.SecurityGroupIds,
			SubnetIds:        eksCluster.ResourcesVpcConfig.SubnetIds,
		}
	}

	apiCluster.Spec.EKSClusterSpec = clusterSpec
	return apiCluster, nil
}

func checkCreatePoolReqValid(machineDeployment apiv2.ExternalClusterMachineDeployment) error {
	eksMD := machineDeployment.Cloud.EKS
	if eksMD == nil {
		return fmt.Errorf("EKS MachineDeployment Spec cannot be nil")
	}

	if len(eksMD.Subnets) == 0 || len(eksMD.NodeRole) == 0 {
		return utilerrors.NewBadRequest("required field is missing: Subnets or NodeRole cannot be empty")
	}
	return nil
}

func createEKSNodePool(cloudSpec *kubermaticv1.ExternalClusterCloudSpec, machineDeployment apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	if err := checkCreatePoolReqValid(machineDeployment); err != nil {
		return nil, err
	}
	eksMD := machineDeployment.Cloud.EKS

	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}
	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return nil, err
	}

	createInput := &eks.CreateNodegroupInput{
		ClusterName:   aws.String(cloudSpec.EKS.Name),
		NodegroupName: aws.String(machineDeployment.Name),
		Subnets:       eksMD.Subnets,
		NodeRole:      aws.String(eksMD.NodeRole),
		AmiType:       aws.String(eksMD.AmiType),
		CapacityType:  aws.String(eksMD.CapacityType),
		DiskSize:      aws.Int64(eksMD.DiskSize),
		InstanceTypes: eksMD.InstanceTypes,
		Labels:        eksMD.Labels,
		ScalingConfig: &eks.NodegroupScalingConfig{
			DesiredSize: aws.Int64(eksMD.ScalingConfig.DesiredSize),
			MaxSize:     aws.Int64(eksMD.ScalingConfig.MaxSize),
			MinSize:     aws.Int64(eksMD.ScalingConfig.MinSize),
		},
	}
	_, err = client.EKS.CreateNodegroup(createInput)
	if err != nil {
		return nil, err
	}

	return &machineDeployment, nil
}

func EKSAMITypesEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var ami types.AMITypes = EKSAMITypes
		var amiTypes apiv2.EKSAMITypes

		for _, amiType := range ami.Values() {
			// AMI type Custom is not valid
			if amiType == EKSCustomAMIType {
				continue
			}
			amiTypes = append(amiTypes, string(amiType))
		}
		return amiTypes, nil
	}
}

func EKSCapacityTypesEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var capacityType types.CapacityTypes = EKSCapacityTypes
		var capacityTypes apiv2.EKSCapacityTypes

		for _, c := range capacityType.Values() {
			capacityTypes = append(capacityTypes, string(c))
		}
		return capacityTypes, nil
	}
}

func deleteEKSCluster(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterCloudSpec) error {
	accessKeyID, secretAccessKey, err := eksprovider.GetCredentialsForCluster(*cloudSpec, secretKeySelector)
	if err != nil {
		return err
	}

	client, err := awsprovider.GetClientSet(accessKeyID, secretAccessKey, "", "", cloudSpec.EKS.Region)
	if err != nil {
		return err
	}

	_, err = client.EKS.DeleteCluster(&eks.DeleteClusterInput{Name: &cloudSpec.EKS.Name})
	if err != nil {
		return err
	}

	return nil
}
