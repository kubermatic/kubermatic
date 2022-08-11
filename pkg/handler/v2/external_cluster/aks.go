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
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/containerservice/mgmt/containerservice"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"
	"github.com/Azure/go-autorest/autorest/to"
	semverlib "github.com/Masterminds/semver/v3"
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
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aks"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/utils/pointer"
)

const (
	AgentPoolModeSystem = "System"
)

// AKSTypesReq represent a request for AKS types.
// swagger:parameters validateAKSCredentials
type AKSTypesReq struct {
	AKSCommonReq
}

// AKSCommonReq represent a request with common parameters for AKS.
type AKSCommonReq struct {
	// in: header
	// name: TenantID
	TenantID string
	// in: header
	// name: SubscriptionID
	SubscriptionID string
	// in: header
	// name: ClientID
	ClientID string
	// in: header
	// name: ClientSecret
	ClientSecret string
	// in: header
	// name: Credential
	Credential string
}

// AKSVMSizesReq represent a request for AKS VM Sizes list.
// swagger:parameters listAKSVMSizes
type AKSVMSizesReq struct {
	AKSTypesReq
	// Location - Resource location
	// in: header
	// name: Location
	Location string
}

// aksNoCredentialReq represent a request for AKS resources
// swagger:parameters listAKSVMSizesNoCredentials
type aksNoCredentialReq struct {
	GetClusterReq
	// Location - Resource location
	// in: header
	// name: Location
	Location string
}

func DecodeAKSTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AKSTypesReq

	commonReq, err := DecodeAKSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.AKSCommonReq = commonReq.(AKSCommonReq)
	return req, nil
}

func DecodeAKSVMSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AKSVMSizesReq

	typesReq, err := DecodeAKSTypesReq(c, r)
	if err != nil {
		return nil, err
	}
	req.AKSTypesReq = typesReq.(AKSTypesReq)
	req.Location = r.Header.Get("Location")

	return req, nil
}

func DecodeAKSNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req aksNoCredentialReq
	re, err := DecodeGetReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = re.(GetClusterReq)
	req.Location = r.Header.Get("Location")

	return req, nil
}

// Validate validates aksNoCredentialReq request.
func (req aksNoCredentialReq) Validate() error {
	if err := req.GetClusterReq.Validate(); err != nil {
		return err
	}
	if len(req.Location) == 0 {
		return fmt.Errorf("AKS Location cannot be empty")
	}
	return nil
}

// Validate validates aksCommonReq request.
func (req AKSVMSizesReq) Validate() error {
	if len(req.Location) == 0 {
		return fmt.Errorf("AKS Location cannot be empty")
	}
	if len(req.Credential) != 0 {
		return nil
	}
	if len(req.TenantID) == 0 || len(req.SubscriptionID) == 0 || len(req.ClientID) == 0 || len(req.ClientSecret) == 0 {
		return fmt.Errorf("AKS credentials cannot be empty")
	}
	return nil
}

func DecodeAKSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AKSCommonReq

	req.TenantID = r.Header.Get("TenantID")
	req.SubscriptionID = r.Header.Get("SubscriptionID")
	req.ClientID = r.Header.Get("ClientID")
	req.ClientSecret = r.Header.Get("ClientSecret")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeAKSClusterListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AKSClusterListReq

	commonReq, err := DecodeAKSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.AKSCommonReq = commonReq.(AKSCommonReq)
	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

// AKSClusterListReq represent a request for AKS cluster list.
// swagger:parameters listAKSClusters
type AKSClusterListReq struct {
	common.ProjectReq
	AKSCommonReq
}

func ListAKSClustersEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(AKSClusterListReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		cred, err := getAKSCredentialsFromReq(ctx, req.AKSCommonReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListAKSClusters(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, clusterProvider, *cred, req.ProjectID)
	}
}

func ListAKSVMSizesEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(AKSVMSizesReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		cred, err := getAKSCredentialsFromReq(ctx, req.AKSCommonReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListAKSVMSizes(ctx, *cred, req.Location)
	}
}

func ListAKSLocationsEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(AKSCommonReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		cred, err := getAKSCredentialsFromReq(ctx, req, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		locations, err := aks.GetLocations(ctx, *cred)
		if err != nil {
			return nil, fmt.Errorf("couldn't get locations: %w", err)
		}

		return locations, nil
	}
}

func AKSNodePoolModesEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var modes apiv2.AKSNodePoolModes

		for _, mode := range containerservice.PossibleAgentPoolModeValues() {
			modes = append(modes, string(mode))
		}
		return modes, nil
	}
}

// Validate validates aksCommonReq request.
func (req AKSCommonReq) Validate() error {
	if len(req.Credential) == 0 && len(req.TenantID) == 0 && len(req.SubscriptionID) == 0 && len(req.ClientID) == 0 && len(req.ClientSecret) == 0 {
		return fmt.Errorf("AKS credentials cannot be empty")
	}
	return nil
}

func AKSValidateCredentialsEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(AKSTypesReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		cred, err := getAKSCredentialsFromReq(ctx, req.AKSCommonReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		err = aks.ValidateCredentials(ctx, *cred)
		if err != nil {
			err = fmt.Errorf("invalid credentials!: %w", err)
		}
		return nil, err
	}
}

func getAKSCredentialsFromReq(ctx context.Context, req AKSCommonReq, userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) (*resources.AKSCredentials, error) {
	subscriptionID := req.SubscriptionID
	clientID := req.ClientID
	clientSecret := req.ClientSecret
	tenantID := req.TenantID

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if len(req.Credential) > 0 {
		preset, err := presetProvider.GetPreset(ctx, userInfo, req.Credential)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
		}
		if credentials := preset.Spec.AKS; credentials != nil {
			subscriptionID = credentials.SubscriptionID
			clientID = credentials.ClientID
			clientSecret = credentials.ClientSecret
			tenantID = credentials.TenantID
		}
	}

	return &resources.AKSCredentials{
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}, nil
}

func createNewAKSCluster(ctx context.Context, aksclusterSpec *apiv2.AKSClusterSpec, aksCloudSpec *apiv2.AKSCloudSpec) error {
	aksClient, err := aks.GetClusterClient(resources.AKSCredentials{
		TenantID:       aksCloudSpec.TenantID,
		ClientID:       aksCloudSpec.ClientID,
		SubscriptionID: aksCloudSpec.SubscriptionID,
		ClientSecret:   aksCloudSpec.ClientSecret,
	})
	if err != nil {
		return err
	}

	clusterSpec := aksclusterSpec
	agentPoolProfiles := clusterSpec.MachineDeploymentSpec
	basicSettings := agentPoolProfiles.BasicSettings
	optionalSettings := agentPoolProfiles.OptionalSettings
	clusterToCreate := armcontainerservice.ManagedCluster{
		Name:     pointer.String(aksCloudSpec.Name),
		Location: pointer.String(clusterSpec.Location),
		Properties: &armcontainerservice.ManagedClusterProperties{
			DNSPrefix:         pointer.String(aksCloudSpec.Name),
			KubernetesVersion: pointer.String(clusterSpec.KubernetesVersion),
			ServicePrincipalProfile: &armcontainerservice.ManagedClusterServicePrincipalProfile{
				ClientID: pointer.String(aksCloudSpec.ClientID),
				Secret:   pointer.String(aksCloudSpec.ClientSecret),
			},
		},
	}

	mode := (armcontainerservice.AgentPoolMode)(basicSettings.Mode)

	azs := []*string{}
	for _, az := range basicSettings.AvailabilityZones {
		azs = append(azs, pointer.String(az))
	}

	agentPoolProfilesToBeCreated := &armcontainerservice.ManagedClusterAgentPoolProfile{
		Name:              pointer.String(agentPoolProfiles.Name),
		VMSize:            pointer.String(basicSettings.VMSize),
		Count:             pointer.Int32(basicSettings.Count),
		Mode:              &mode,
		OSDiskSizeGB:      pointer.Int32(basicSettings.OsDiskSizeGB),
		AvailabilityZones: azs,
		NodeLabels:        optionalSettings.NodeLabels,
	}

	if basicSettings.EnableAutoScaling {
		agentPoolProfilesToBeCreated.EnableAutoScaling = to.BoolPtr(basicSettings.EnableAutoScaling)
		agentPoolProfilesToBeCreated.MaxCount = pointer.Int32(basicSettings.ScalingConfig.MaxCount)
		agentPoolProfilesToBeCreated.MinCount = pointer.Int32(basicSettings.ScalingConfig.MinCount)
	}

	clusterToCreate.Properties.AgentPoolProfiles = []*armcontainerservice.ManagedClusterAgentPoolProfile{
		agentPoolProfilesToBeCreated,
	}

	_, err = aksClient.BeginCreateOrUpdate(
		ctx,
		aksCloudSpec.ResourceGroup,
		aksCloudSpec.Name,
		clusterToCreate,
		nil,
	)

	return aks.DecodeError(err)
}

func checkCreatePoolReqValidity(aksMD *apiv2.AKSMachineDeploymentCloudSpec) error {
	if aksMD == nil {
		return utilerrors.NewBadRequest("AKS MachineDeploymentSpec cannot be nil")
	}
	basicSettings := aksMD.BasicSettings
	// check whether required fields for nodepool creation are provided
	fields := reflect.ValueOf(&basicSettings).Elem()
	for i := 0; i < fields.NumField(); i++ {
		yourjsonTags := fields.Type().Field(i).Tag.Get("required")
		if strings.Contains(yourjsonTags, "true") && fields.Field(i).IsZero() {
			return utilerrors.NewBadRequest("required field is missing: %v", fields.Type().Field(i).Name)
		}
	}
	if basicSettings.EnableAutoScaling {
		maxCount := basicSettings.ScalingConfig.MaxCount
		minCount := basicSettings.ScalingConfig.MinCount
		if maxCount == 0 {
			return utilerrors.NewBadRequest("InvalidParameter: value of maxCount for enabled autoscaling is invalid")
		}
		if minCount == 0 {
			return utilerrors.NewBadRequest("InvalidParameter: value of minCount for enabled autoscaling is invalid")
		}
	}
	return nil
}

func checkCreateClusterReqValidity(aksclusterSpec *apiv2.AKSClusterSpec) error {
	if len(aksclusterSpec.Location) == 0 {
		return utilerrors.NewBadRequest("required field is missing: Location")
	}
	agentPoolProfiles := aksclusterSpec.MachineDeploymentSpec
	if agentPoolProfiles == nil || agentPoolProfiles.BasicSettings.Mode != AgentPoolModeSystem {
		return utilerrors.NewBadRequest("Must define at least one system pool!")
	}
	return checkCreatePoolReqValidity(agentPoolProfiles)
}

func createOrImportAKSCluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticv1.Project, spec *apiv2.ExternalClusterSpec, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticv1.ExternalCluster, error) {
	isImported := resources.ExternalClusterIsImportedTrue

	// check whether required fields for cluster import are provided
	fields := reflect.ValueOf(cloud.AKS).Elem()
	for i := 0; i < fields.NumField(); i++ {
		yourjsonTags := fields.Type().Field(i).Tag.Get("required")
		if strings.Contains(yourjsonTags, "true") && fields.Field(i).IsZero() {
			return nil, utilerrors.NewBadRequest("required field is missing: %v", fields.Type().Field(i).Name)
		}
	}

	if spec != nil && spec.AKSClusterSpec != nil {
		if err := checkCreateClusterReqValidity(spec.AKSClusterSpec); err != nil {
			return nil, err
		}
		if err := createNewAKSCluster(ctx, spec.AKSClusterSpec, cloud.AKS); err != nil {
			return nil, err
		}
		isImported = resources.ExternalClusterIsImportedFalse
	}
	newCluster := genExternalCluster(name, project.Name, isImported)
	newCluster.Spec.CloudSpec = &kubermaticv1.ExternalClusterCloudSpec{
		AKS: &kubermaticv1.ExternalClusterAKSCloudSpec{
			Name:          cloud.AKS.Name,
			ResourceGroup: cloud.AKS.ResourceGroup,
		},
	}
	keyRef, err := clusterProvider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, project.Name, newCluster.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	newCluster.Spec.CloudSpec.AKS.CredentialsReference = keyRef

	return createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
}

func patchAKSCluster(ctx context.Context, oldCluster, newCluster *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterAKSCloudSpec) (*apiv2.ExternalCluster, error) {
	clusterName := cloudSpec.Name
	resourceGroup := cloudSpec.ResourceGroup

	newVersion := newCluster.Spec.Version.Semver().String()

	cred, err := aks.GetCredentialsForCluster(cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := aks.GetClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := aks.GetCluster(ctx, aksClient, cloudSpec)
	if err != nil {
		return nil, err
	}

	location := aksCluster.Location

	updateCluster := armcontainerservice.ManagedCluster{
		Location: location,
		Properties: &armcontainerservice.ManagedClusterProperties{
			KubernetesVersion: &newVersion,
		},
	}

	_, err = aksClient.BeginCreateOrUpdate(ctx, resourceGroup, clusterName, updateCluster, nil)
	if err != nil {
		return nil, aks.DecodeError(err)
	}

	return newCluster, nil
}

func getAKSNodePools(ctx context.Context, cluster *kubermaticv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	cloud := cluster.Spec.CloudSpec

	cred, err := aks.GetCredentialsForCluster(cloud.AKS, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := aks.GetClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := aks.GetCluster(ctx, aksClient, cloud.AKS)
	if err != nil {
		return nil, err
	}

	poolProfiles := aksCluster.Properties.AgentPoolProfiles

	return getAKSMachineDeployments(ctx, poolProfiles, cluster, clusterProvider)
}

func getAKSMachineDeployments(ctx context.Context, poolProfiles []*armcontainerservice.ManagedClusterAgentPoolProfile, cluster *kubermaticv1.ExternalCluster, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	machineDeployments := make([]apiv2.ExternalClusterMachineDeployment, 0, len(poolProfiles))

	nodes, err := clusterProvider.ListNodes(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	for _, poolProfile := range poolProfiles {
		readyReplicasCount := kuberneteshelper.GetNodeGroupReadyCount(nodes, resources.AKSNodepoolNameLabel, *poolProfile.Name)
		machineDeployments = append(machineDeployments, createMachineDeploymentFromAKSNodePoll(poolProfile, readyReplicasCount))
	}

	return machineDeployments, nil
}

func getAKSNodePool(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodePoolName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	cloud := cluster.Spec.CloudSpec

	cred, err := aks.GetCredentialsForCluster(cloud.AKS, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := aks.GetClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := aks.GetCluster(ctx, aksClient, cloud.AKS)
	if err != nil {
		return nil, err
	}

	var poolProfile *armcontainerservice.ManagedClusterAgentPoolProfile
	for _, agentPoolProperty := range aksCluster.Properties.AgentPoolProfiles {
		if *agentPoolProperty.Name == nodePoolName {
			poolProfile = agentPoolProperty
			break
		}
	}

	if poolProfile == nil {
		return nil, fmt.Errorf("no nodePool found with the name: %v", nodePoolName)
	}

	return getAKSMachineDeployment(ctx, poolProfile, cluster, clusterProvider)
}

func getAKSMachineDeployment(ctx context.Context, poolProfile *armcontainerservice.ManagedClusterAgentPoolProfile, cluster *kubermaticv1.ExternalCluster, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	nodes, err := clusterProvider.ListNodes(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	readyReplicasCount := kuberneteshelper.GetNodeGroupReadyCount(nodes, resources.AKSNodepoolNameLabel, *poolProfile.Name)

	md := createMachineDeploymentFromAKSNodePoll(poolProfile, readyReplicasCount)
	return &md, nil
}

func createMachineDeploymentFromAKSNodePoll(nodePool *armcontainerservice.ManagedClusterAgentPoolProfile, readyReplicas int32) apiv2.ExternalClusterMachineDeployment {
	name := to.String(nodePool.Name)
	md := apiv2.ExternalClusterMachineDeployment{
		NodeDeployment: apiv1.NodeDeployment{
			ObjectMeta: apiv1.ObjectMeta{
				ID:   name,
				Name: name,
			},
			Spec: apiv1.NodeDeploymentSpec{
				Replicas: to.Int32(nodePool.Count),
				Template: apiv1.NodeSpec{
					Versions: apiv1.NodeVersionInfo{
						Kubelet: to.String(nodePool.OrchestratorVersion),
					},
				},
			},
			Status: clusterv1alpha1.MachineDeploymentStatus{
				Replicas:      to.Int32(nodePool.Count),
				ReadyReplicas: readyReplicas,
			},
		},
		Cloud: &apiv2.ExternalClusterMachineDeploymentCloudSpec{
			AKS: &apiv2.AKSMachineDeploymentCloudSpec{},
		},
	}

	azs := []string{}
	for _, az := range nodePool.AvailabilityZones {
		azs = append(azs, *az)
	}

	md.Cloud.AKS.Name = name
	md.Cloud.AKS.BasicSettings = apiv2.AgentPoolBasics{
		Mode:                string(*nodePool.Mode),
		AvailabilityZones:   azs,
		OrchestratorVersion: to.String(nodePool.OrchestratorVersion),
		VMSize:              to.String(nodePool.VMSize),
		EnableAutoScaling:   to.Bool(nodePool.EnableAutoScaling),
		Count:               to.Int32(nodePool.Count),
		OsDiskSizeGB:        to.Int32(nodePool.OSDiskSizeGB),
	}
	if md.Cloud.AKS.BasicSettings.EnableAutoScaling {
		md.Cloud.AKS.BasicSettings.ScalingConfig.MaxCount = to.Int32(nodePool.MaxCount)
		md.Cloud.AKS.BasicSettings.ScalingConfig.MinCount = to.Int32(nodePool.MinCount)
	}
	md.Cloud.AKS.Configuration = apiv2.AgentPoolConfig{
		OsType:             string(*nodePool.OSType),
		OsDiskType:         string(*nodePool.OSDiskType),
		VnetSubnetID:       to.String(nodePool.VnetSubnetID),
		PodSubnetID:        to.String(nodePool.VnetSubnetID),
		MaxPods:            to.Int32(nodePool.MaxPods),
		EnableNodePublicIP: to.Bool(nodePool.EnableNodePublicIP),
	}

	taints := []string{}
	for _, t := range nodePool.NodeTaints {
		taints = append(taints, *t)
	}

	md.Cloud.AKS.OptionalSettings = apiv2.AgentPoolOptionalSettings{
		NodeLabels: nodePool.NodeLabels,
		NodeTaints: taints,
	}
	if nodePool.UpgradeSettings != nil {
		md.Cloud.AKS.Configuration.MaxSurgeUpgradeSetting = to.String(nodePool.UpgradeSettings.MaxSurge)
	}
	if nodePool.ProvisioningState != nil && (nodePool.PowerState != nil || nodePool.PowerState.Code != nil) {
		md.Phase = apiv2.ExternalClusterMDPhase{
			State: aks.ConvertMDStatus(*nodePool.ProvisioningState, *nodePool.PowerState.Code),
		}
	}

	return md
}

func patchAKSMachineDeployment(ctx context.Context, oldCluster, newCluster *apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, cloud *kubermaticv1.ExternalClusterAKSCloudSpec) (*apiv2.ExternalClusterMachineDeployment, error) {
	cred, err := aks.GetCredentialsForCluster(cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	agentPoolClient, err := getAKSNodePoolClient(cred)
	if err != nil {
		return nil, err
	}

	nodePoolName := newCluster.NodeDeployment.Name
	currentReplicas := oldCluster.NodeDeployment.Spec.Replicas
	desiredReplicas := newCluster.NodeDeployment.Spec.Replicas
	currentVersion := oldCluster.NodeDeployment.Spec.Template.Versions.Kubelet
	desiredVersion := newCluster.NodeDeployment.Spec.Template.Versions.Kubelet
	if desiredReplicas != currentReplicas {
		_, err = resizeAKSNodePool(ctx, *agentPoolClient, cloud, nodePoolName, desiredReplicas)
		if err != nil {
			return nil, err
		}
		newCluster.NodeDeployment.Status.Replicas = desiredReplicas
		return newCluster, nil
	}
	if desiredVersion != currentVersion {
		_, err = upgradeNodePool(ctx, *agentPoolClient, cloud, nodePoolName, desiredVersion)
		if err != nil {
			return nil, err
		}
		newCluster.NodeDeployment.Spec.Replicas = currentReplicas
		return newCluster, nil
	}

	return newCluster, nil
}

func resizeAKSNodePool(ctx context.Context, agentPoolClient armcontainerservice.AgentPoolsClient, cloud *kubermaticv1.ExternalClusterAKSCloudSpec, nodePoolName string, desiredSize int32) (*runtime.Poller[armcontainerservice.AgentPoolsClientCreateOrUpdateResponse], error) {
	pool, err := agentPoolClient.Get(ctx, cloud.ResourceGroup, cloud.Name, nodePoolName, nil)
	if err != nil {
		return nil, err
	}
	nodePool := armcontainerservice.AgentPool{
		Name: &nodePoolName,
		Properties: &armcontainerservice.ManagedClusterAgentPoolProfileProperties{
			Count: &desiredSize,
		},
	}

	if pool.Properties.Mode != nil && *pool.Properties.Mode == armcontainerservice.AgentPoolModeSystem {
		nodePool.Properties.Mode = pool.Properties.Mode
	}

	return updateAKSNodePool(ctx, agentPoolClient, cloud, nodePoolName, nodePool)
}

func upgradeNodePool(ctx context.Context, agentPoolClient armcontainerservice.AgentPoolsClient, cloud *kubermaticv1.ExternalClusterAKSCloudSpec, nodePoolName string, desiredVersion string) (*runtime.Poller[armcontainerservice.AgentPoolsClientCreateOrUpdateResponse], error) {
	pool, err := agentPoolClient.Get(ctx, cloud.ResourceGroup, cloud.Name, nodePoolName, nil)
	if err != nil {
		return nil, err
	}
	nodePool := armcontainerservice.AgentPool{
		Name: &nodePoolName,
		Properties: &armcontainerservice.ManagedClusterAgentPoolProfileProperties{
			OrchestratorVersion: &desiredVersion,
		},
	}

	if pool.Properties.Mode != nil && *pool.Properties.Mode == armcontainerservice.AgentPoolModeSystem {
		nodePool.Properties.Mode = pool.Properties.Mode
	}

	return updateAKSNodePool(ctx, agentPoolClient, cloud, nodePoolName, nodePool)
}

func getAKSNodePoolClient(cred resources.AKSCredentials) (*armcontainerservice.AgentPoolsClient, error) {
	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return nil, err
	}
	agentPoolClient, err := armcontainerservice.NewAgentPoolsClient(cred.SubscriptionID, azcred, nil)
	return agentPoolClient, aks.DecodeError(err)
}

func updateAKSNodePool(ctx context.Context, agentPoolClient armcontainerservice.AgentPoolsClient, cloud *kubermaticv1.ExternalClusterAKSCloudSpec, nodePoolName string, nodePool armcontainerservice.AgentPool) (*runtime.Poller[armcontainerservice.AgentPoolsClientCreateOrUpdateResponse], error) {
	return agentPoolClient.BeginCreateOrUpdate(ctx, cloud.ResourceGroup, cloud.Name, nodePoolName, nodePool, nil)
}

func deleteAKSNodeGroup(ctx context.Context, cloud *kubermaticv1.ExternalClusterAKSCloudSpec, nodePoolName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) error {
	cred, err := aks.GetCredentialsForCluster(cloud, secretKeySelector)
	if err != nil {
		return err
	}

	agentPoolClient, err := getAKSNodePoolClient(cred)
	if err != nil {
		return err
	}

	future, err := agentPoolClient.BeginDelete(ctx, cloud.ResourceGroup, cloud.Name, nodePoolName, nil)
	if err != nil {
		return err
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}

func createAKSNodePool(ctx context.Context, cloud *kubermaticv1.ExternalClusterAKSCloudSpec, machineDeployment apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	cred, err := aks.GetCredentialsForCluster(cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	agentPoolClient, err := getAKSNodePoolClient(cred)
	if err != nil {
		return nil, err
	}

	nodePool := &armcontainerservice.AgentPool{
		Name: &machineDeployment.Name,
	}

	aksMD := machineDeployment.Cloud.AKS
	if err := checkCreatePoolReqValidity(aksMD); err != nil {
		return nil, err
	}
	basicSettings := aksMD.BasicSettings
	optionalSettings := aksMD.OptionalSettings

	taints := []*string{}
	for i := range optionalSettings.NodeTaints {
		taints = append(taints, &optionalSettings.NodeTaints[i])
	}

	azs := []*string{}
	for i := range basicSettings.AvailabilityZones {
		azs = append(azs, &basicSettings.AvailabilityZones[i])
	}

	mode := (armcontainerservice.AgentPoolMode)(basicSettings.Mode)

	property := armcontainerservice.ManagedClusterAgentPoolProfileProperties{
		VMSize:              to.StringPtr(basicSettings.VMSize),
		Count:               to.Int32Ptr(basicSettings.Count),
		OrchestratorVersion: to.StringPtr(basicSettings.OrchestratorVersion),
		Mode:                &mode,
		OSDiskSizeGB:        to.Int32Ptr(basicSettings.OsDiskSizeGB),
		AvailabilityZones:   azs,
		EnableAutoScaling:   to.BoolPtr(basicSettings.EnableAutoScaling),
		NodeLabels:          optionalSettings.NodeLabels,
		NodeTaints:          taints,
	}
	if basicSettings.EnableAutoScaling {
		property.MaxCount = to.Int32Ptr(basicSettings.ScalingConfig.MaxCount)
		property.MinCount = to.Int32Ptr(basicSettings.ScalingConfig.MinCount)
	}
	nodePool.Properties = &property

	_, err = agentPoolClient.BeginCreateOrUpdate(ctx, cloud.ResourceGroup, cloud.Name, *nodePool.Name, *nodePool, nil)
	if err != nil {
		return nil, aks.DecodeError(err)
	}

	machineDeployment.Phase = apiv2.ExternalClusterMDPhase{
		State: apiv2.ProvisioningExternalClusterMDState,
	}

	return &machineDeployment, nil
}

func AKSNodeVersionsWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req, ok := request.(GetClusterReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
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
		cloud := cluster.Spec.CloudSpec
		if cloud.AKS == nil {
			return nil, utilerrors.NewNotFound("cloud spec for %s", req.ClusterID)
		}

		resourceGroup := cloud.AKS.ResourceGroup
		resourceName := cloud.AKS.Name
		availableVersions := make([]*apiv1.MasterVersion, 0)

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
		cred, err := aks.GetCredentialsForCluster(cloud.AKS, secretKeySelector)
		if err != nil {
			return nil, err
		}

		agentPoolClient, err := getAKSNodePoolClient(cred)
		if err != nil {
			return nil, err
		}

		agentPoolAvailableVersions, err := agentPoolClient.GetAvailableAgentPoolVersions(ctx, resourceGroup, resourceName, nil)
		if err != nil {
			return nil, err
		}

		properties := agentPoolAvailableVersions.Properties
		if properties == nil || properties.AgentPoolVersions == nil {
			return availableVersions, nil
		}

		for _, version := range properties.AgentPoolVersions {
			if version.KubernetesVersion != nil {
				kubernetesVersion, err := semverlib.NewVersion(to.String(version.KubernetesVersion))
				if err != nil {
					return nil, err
				}
				availableVersions = append(availableVersions, &apiv1.MasterVersion{Version: kubernetesVersion, Default: to.Bool(version.Default)})
			}
		}

		return availableVersions, nil
	}
}

func AKSSizesWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(aksNoCredentialReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
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
		cloud := cluster.Spec.CloudSpec
		if cloud.AKS == nil {
			return nil, utilerrors.NewNotFound("cloud spec for %s", req.ClusterID)
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
		cred, err := aks.GetCredentialsForCluster(cloud.AKS, secretKeySelector)
		if err != nil {
			return nil, err
		}

		return providercommon.ListAKSVMSizes(ctx, cred, req.Location)
	}
}

func getAKSClusterDetails(ctx context.Context, apiCluster *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloud *kubermaticv1.ExternalClusterAKSCloudSpec) (*apiv2.ExternalCluster, error) {
	cred, err := aks.GetCredentialsForCluster(cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	aksClient, err := aks.GetClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := aks.GetCluster(ctx, aksClient, cloud)
	if err != nil {
		return nil, err
	}

	aksClusterProperties := aksCluster.Properties
	if aksClusterProperties == nil {
		return apiCluster, nil
	}
	clusterSpec := &apiv2.AKSClusterSpec{
		Location:          to.String(aksCluster.Location),
		Tags:              aksCluster.Tags,
		DNSPrefix:         to.String(aksClusterProperties.DNSPrefix),
		KubernetesVersion: to.String(aksClusterProperties.KubernetesVersion),
		EnableRBAC:        to.Bool(aksClusterProperties.EnableRBAC),
		FqdnSubdomain:     to.String(aksClusterProperties.FqdnSubdomain),
		Fqdn:              to.String(aksClusterProperties.Fqdn),
		PrivateFQDN:       to.String(aksClusterProperties.PrivateFQDN),
	}

	if aksCluster.SystemData != nil {
		clusterSpec.CreatedAt = aksCluster.SystemData.CreatedAt
		clusterSpec.CreatedBy = aksCluster.SystemData.CreatedBy
	}
	networkProfile := aksClusterProperties.NetworkProfile
	if networkProfile != nil {
		clusterSpec.NetworkProfile = apiv2.AKSNetworkProfile{
			PodCidr:          to.String(networkProfile.PodCidr),
			ServiceCidr:      to.String(networkProfile.ServiceCidr),
			DNSServiceIP:     to.String(networkProfile.DNSServiceIP),
			DockerBridgeCidr: to.String(networkProfile.DockerBridgeCidr),
		}
	}
	if networkProfile.NetworkPlugin != nil {
		clusterSpec.NetworkProfile.NetworkPlugin = string(*networkProfile.NetworkPlugin)
	}
	if networkProfile.NetworkPolicy != nil {
		clusterSpec.NetworkProfile.NetworkPolicy = string(*networkProfile.NetworkPolicy)
	}
	if networkProfile.NetworkMode != nil {
		clusterSpec.NetworkProfile.NetworkMode = string(*networkProfile.NetworkMode)
	}
	apiCluster.Spec.AKSClusterSpec = clusterSpec

	return apiCluster, nil
}

func deleteAKSCluster(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloud *kubermaticv1.ExternalClusterAKSCloudSpec) error {
	cred, err := aks.GetCredentialsForCluster(cloud, secretKeySelector)
	if err != nil {
		return err
	}
	aksClient, err := aks.GetClusterClient(cred)
	if err != nil {
		return err
	}
	err = aks.DeleteCluster(ctx, aksClient, cloud)
	if err != nil {
		return err
	}

	return nil
}

func AKSVersionsEndpoint(configGetter provider.KubermaticConfigurationGetter,
	clusterProvider provider.ExternalClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return clusterProvider.VersionsEndpoint(ctx, configGetter, kubermaticv1.AKSProviderType)
	}
}
