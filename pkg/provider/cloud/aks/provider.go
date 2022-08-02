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
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"
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

func GetCredentialsForCluster(cloud kubermaticv1.ExternalClusterCloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (resources.AKSCredentials, error) {
	tenantID := cloud.AKS.TenantID
	subscriptionID := cloud.AKS.SubscriptionID
	clientID := cloud.AKS.ClientID
	clientSecret := cloud.AKS.ClientSecret
	cred := resources.AKSCredentials{}
	var err error

	if tenantID == "" {
		if cloud.AKS.CredentialsReference == nil {
			return cred, errors.New("no credentials provided")
		}
		tenantID, err = secretKeySelector(cloud.AKS.CredentialsReference, resources.AzureTenantID)
		if err != nil {
			return cred, err
		}
	}

	if subscriptionID == "" {
		if cloud.AKS.CredentialsReference == nil {
			return cred, errors.New("no credentials provided")
		}
		subscriptionID, err = secretKeySelector(cloud.AKS.CredentialsReference, resources.AzureSubscriptionID)
		if err != nil {
			return cred, err
		}
	}

	if clientID == "" {
		if cloud.AKS.CredentialsReference == nil {
			return cred, errors.New("no credentials provided")
		}
		clientID, err = secretKeySelector(cloud.AKS.CredentialsReference, resources.AzureClientID)
		if err != nil {
			return cred, err
		}
	}

	if clientSecret == "" {
		if cloud.AKS.CredentialsReference == nil {
			return cred, errors.New("no credentials provided")
		}
		clientSecret, err = secretKeySelector(cloud.AKS.CredentialsReference, resources.AzureClientSecret)
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

func GetCluster(ctx context.Context, aksClient *armcontainerservice.ManagedClustersClient, cloud *kubermaticv1.ExternalClusterCloudSpec) (*armcontainerservice.ManagedCluster, error) {
	aksCluster, err := aksClient.Get(ctx, cloud.AKS.ResourceGroup, cloud.AKS.Name, nil)
	if err != nil {
		return nil, DecodeError(err)
	}

	return &aksCluster.ManagedCluster, nil
}

func GetClusterStatus(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloud *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalClusterStatus, error) {
	cred, err := GetCredentialsForCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := GetClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := GetCluster(ctx, aksClient, cloud)
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

func DeleteCluster(ctx context.Context, aksClient *armcontainerservice.ManagedClustersClient, cloud *kubermaticv1.ExternalClusterCloudSpec) error {
	resourceGroup := cloud.AKS.ResourceGroup
	clusterName := cloud.AKS.Name

	_, err := aksClient.BeginDelete(ctx, resourceGroup, clusterName, &armcontainerservice.ManagedClustersClientBeginDeleteOptions{})
	return DecodeError(err)
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

func DecodeError(err error) error {
	if err == nil {
		return nil
	}
	var aerr *azcore.ResponseError
	if ok := errors.As(err, &aerr); ok {
		type response struct {
			Code    string `json:"code,omitempty"`
			Message string `json:"message,omitempty"`
			SubCode string `json:"subcode,omitempty"`
		}
		responsemap := map[string]response{}
		code := aerr.StatusCode
		if err := json.NewDecoder(aerr.RawResponse.Body).Decode(&responsemap); err != nil {
			return err
		}

		return utilerrors.New(code, responsemap["error"].Message)
	}
	return err
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
