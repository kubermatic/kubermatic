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

package aks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/Azure/go-autorest/autorest/to"
	semverlib "github.com/Masterminds/semver/v3"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func GetLocations(ctx context.Context, cred resources.AKSCredentials) (apiv2.AKSLocationList, error) {
	var locationList apiv2.AKSLocationList
	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return nil, DecodeError(err)
	}
	client, err := armsubscriptions.NewClient(azcred, &arm.ClientOptions{})
	if err != nil {
		return nil, DecodeError(err)
	}

	pager := client.NewListLocationsPager(cred.SubscriptionID, &armsubscriptions.ClientListLocationsOptions{
		IncludeExtendedLocations: to.BoolPtr(false),
	})
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, DecodeError(err)
		}
		for _, v := range nextResult.Value {
			if v.Name != nil && v.Metadata != nil && v.Metadata.RegionCategory != nil {
				if *v.Metadata.RegionCategory == "Recommended" && *v.Name != "eastus2euap" {
					locationList = append(locationList, apiv2.AKSLocation{
						Name:           *v.Name,
						RegionCategory: string(*v.Metadata.RegionCategory),
					},
					)
				}
			}
		}
	}
	return locationList, nil
}

func GetClusterConfig(ctx context.Context, cred resources.AKSCredentials, clusterName, resourceGroupName string) (*api.Config, error) {
	aksClient, err := GetClusterClient(cred)
	if err != nil {
		return nil, err
	}

	credResult, err := aksClient.ListClusterAdminCredentials(ctx, resourceGroupName, clusterName, nil)
	if err != nil {
		return nil, DecodeError(err)
	}

	config, err := clientcmd.Load(credResult.Kubeconfigs[0].Value)
	if err != nil {
		return nil, fmt.Errorf("cannot get azure cluster config: %w", err)
	}

	return config, nil
}

func GetCredentialsForCluster(cloud *kubermaticv1.ExternalClusterAKSCloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (resources.AKSCredentials, error) {
	tenantID := cloud.TenantID
	subscriptionID := cloud.SubscriptionID
	clientID := cloud.ClientID
	clientSecret := cloud.ClientSecret
	cred := resources.AKSCredentials{}
	var err error

	if tenantID == "" {
		if cloud.CredentialsReference == nil {
			return cred, errors.New("no credentials provided")
		}
		tenantID, err = secretKeySelector(cloud.CredentialsReference, resources.AzureTenantID)
		if err != nil {
			return cred, err
		}
	}

	if subscriptionID == "" {
		if cloud.CredentialsReference == nil {
			return cred, errors.New("no credentials provided")
		}
		subscriptionID, err = secretKeySelector(cloud.CredentialsReference, resources.AzureSubscriptionID)
		if err != nil {
			return cred, err
		}
	}

	if clientID == "" {
		if cloud.CredentialsReference == nil {
			return cred, errors.New("no credentials provided")
		}
		clientID, err = secretKeySelector(cloud.CredentialsReference, resources.AzureClientID)
		if err != nil {
			return cred, err
		}
	}

	if clientSecret == "" {
		if cloud.CredentialsReference == nil {
			return cred, errors.New("no credentials provided")
		}
		clientSecret, err = secretKeySelector(cloud.CredentialsReference, resources.AzureClientSecret)
		if err != nil {
			return cred, err
		}
	}

	cred = resources.AKSCredentials{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}
	return cred, nil
}

func GetClusterClient(cred resources.AKSCredentials) (*armcontainerservice.ManagedClustersClient, error) {
	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	client, err := armcontainerservice.NewManagedClustersClient(cred.SubscriptionID, azcred, nil)
	return client, DecodeError(err)
}

func GetCluster(ctx context.Context, aksClient *armcontainerservice.ManagedClustersClient, cloud *kubermaticv1.ExternalClusterAKSCloudSpec) (*armcontainerservice.ManagedCluster, error) {
	aksCluster, err := aksClient.Get(ctx, cloud.ResourceGroup, cloud.Name, nil)
	if err != nil {
		return nil, DecodeError(err)
	}

	return &aksCluster.ManagedCluster, nil
}

func GetClusterStatus(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterAKSCloudSpec) (*apiv2.ExternalClusterStatus, error) {
	cred, err := GetCredentialsForCluster(cloudSpec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := GetClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := GetCluster(ctx, aksClient, cloudSpec)
	if err != nil {
		return nil, err
	}
	state := apiv2.UnknownExternalClusterState
	if aksCluster.Properties != nil {
		var powerState armcontainerservice.Code
		var provisioningState string
		if aksCluster.Properties.PowerState != nil {
			powerState = *aksCluster.Properties.PowerState.Code
		}
		if aksCluster.Properties.ProvisioningState != nil {
			provisioningState = *aksCluster.Properties.ProvisioningState
		}
		state = ConvertStatus(provisioningState, powerState)
	}

	return &apiv2.ExternalClusterStatus{
		State: state,
	}, nil
}

func DeleteCluster(ctx context.Context, aksClient *armcontainerservice.ManagedClustersClient, cloudSpec *kubermaticv1.ExternalClusterAKSCloudSpec) error {
	resourceGroup := cloudSpec.ResourceGroup
	clusterName := cloudSpec.Name

	_, err := aksClient.BeginDelete(ctx, resourceGroup, clusterName, &armcontainerservice.ManagedClustersClientBeginDeleteOptions{})
	return DecodeError(err)
}

func ConvertStatus(provisioningState string, powerState armcontainerservice.Code) apiv2.ExternalClusterState {
	switch {
	case provisioningState == string(resources.CreatingAKSState):
		return apiv2.ProvisioningExternalClusterState
	case provisioningState == string(resources.SucceededAKSState) && powerState == armcontainerservice.Code(resources.RunningAKSState):
		return apiv2.RunningExternalClusterState
	case provisioningState == string(resources.StartingAKSState):
		return apiv2.ProvisioningExternalClusterState
	case provisioningState == string(resources.StoppingAKSState):
		return apiv2.StoppingExternalClusterState
	case provisioningState == string(resources.SucceededAKSState) && powerState == armcontainerservice.Code(resources.StoppedAKSState):
		return apiv2.StoppedExternalClusterState
	case provisioningState == string(resources.FailedAKSState):
		return apiv2.ErrorExternalClusterState
	case provisioningState == string(resources.DeletingAKSState):
		return apiv2.DeletingExternalClusterState
	case provisioningState == string(resources.UpgradingAKSState):
		return apiv2.ReconcilingExternalClusterState
	default:
		return apiv2.UnknownExternalClusterState
	}
}

func ConvertMDStatus(provisioningState string, powerState armcontainerservice.Code) apiv2.ExternalClusterMDState {
	switch {
	case provisioningState == string(resources.CreatingAKSMDState):
		return apiv2.ProvisioningExternalClusterMDState
	case provisioningState == string(resources.SucceededAKSMDState) && string(powerState) == string(resources.RunningAKSMDState):
		return apiv2.RunningExternalClusterMDState
	case provisioningState == string(resources.FailedAKSMDState):
		return apiv2.ErrorExternalClusterMDState
	case provisioningState == string(resources.DeletingAKSMDState):
		return apiv2.DeletingExternalClusterMDState
	// "Upgrading" indicates Kubernetes version upgrade.
	// "Updating" indicates MachineDeployment Replica Scale.
	case sets.NewString(string(resources.UpgradingAKSMDState), string(resources.UpdatingAKSMDState), string(resources.ScalingAKSMDState)).Has(provisioningState):
		return apiv2.ReconcilingExternalClusterMDState
	default:
		return apiv2.UnknownExternalClusterMDState
	}
}

func ListMachineDeploymentUpgrades(ctx context.Context, cred resources.AKSCredentials, clusterName, resourceGroupName, machineDeployment string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)

	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	agentPoolClient, err := armcontainerservice.NewAgentPoolsClient(cred.SubscriptionID, azcred, nil)
	if err != nil {
		return nil, DecodeError(err)
	}

	profile, err := agentPoolClient.GetUpgradeProfile(ctx, resourceGroupName, clusterName, machineDeployment, nil)
	if err != nil {
		return nil, DecodeError(err)
	}

	poolUpgradeProperties := profile.Properties
	if poolUpgradeProperties == nil || poolUpgradeProperties.Upgrades == nil {
		return upgrades, nil
	}

	for _, poolUpgrade := range poolUpgradeProperties.Upgrades {
		if poolUpgrade.KubernetesVersion != nil {
			upgradeMachineDeploymentVer, err := semverlib.NewVersion(*poolUpgrade.KubernetesVersion)
			if err != nil {
				return nil, err
			}
			upgrades = append(upgrades, &apiv1.MasterVersion{Version: upgradeMachineDeploymentVer})
		}
	}

	return upgrades, nil
}

func ListUpgrades(ctx context.Context, cred resources.AKSCredentials, resourceGroupName, resourceName string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)

	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	aksClient, err := armcontainerservice.NewManagedClustersClient(cred.SubscriptionID, azcred, nil)
	if err != nil {
		return nil, DecodeError(err)
	}

	clusterUpgradeProfile, err := aksClient.GetUpgradeProfile(ctx, resourceGroupName, resourceName, nil)
	if err != nil {
		return nil, DecodeError(err)
	}

	upgradeProperties := clusterUpgradeProfile.Properties
	if upgradeProperties == nil || upgradeProperties.ControlPlaneProfile == nil || upgradeProperties.ControlPlaneProfile.Upgrades == nil {
		return upgrades, nil
	}

	for _, upgradesItem := range upgradeProperties.ControlPlaneProfile.Upgrades {
		v, err := ksemver.NewSemver(*upgradesItem.KubernetesVersion)
		if err != nil {
			return nil, err
		}
		upgrades = append(upgrades, &apiv1.MasterVersion{
			Version: v.Semver(),
		})
	}
	return upgrades, nil
}

func DecodeError(err error) error {
	var aerr *azcore.ResponseError
	if errors.As(err, &aerr) {
		type response struct {
			Code    string `json:"code,omitempty"`
			Message string `json:"message,omitempty"`
			SubCode string `json:"subcode,omitempty"`
		}
		responseMap := map[string]response{}
		if err := json.NewDecoder(aerr.RawResponse.Body).Decode(&responseMap); err != nil {
			return err
		}
		return utilerrors.New(aerr.StatusCode, responseMap["error"].Message)
	}
	return err
}
