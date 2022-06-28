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
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func GetClusterConfig(ctx context.Context, cred resources.AKSCredentials, clusterName, resourceGroupName string) (*api.Config, error) {
	aksClient, err := GetAKSClusterClient(cred)
	if err != nil {
		return nil, err
	}

	credResult, err := aksClient.ListClusterAdminCredentials(ctx, resourceGroupName, clusterName, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get azure cluster config: %w", err)
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

func GetAKSClusterClient(cred resources.AKSCredentials) (*armcontainerservice.ManagedClustersClient, error) {
	azcred, err := azidentity.NewClientSecretCredential(cred.TenantID, cred.ClientID, cred.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	return armcontainerservice.NewManagedClustersClient(cred.SubscriptionID, azcred, nil)
}

func GetAKSCluster(ctx context.Context, aksClient *armcontainerservice.ManagedClustersClient, cloud *kubermaticv1.ExternalClusterCloudSpec) (*armcontainerservice.ManagedCluster, error) {
	resourceGroup := cloud.AKS.ResourceGroup
	clusterName := cloud.AKS.Name

	aksCluster, err := aksClient.Get(ctx, cloud.AKS.ResourceGroup, cloud.AKS.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get AKS managed cluster %v from resource group %v: %w", clusterName, resourceGroup, err)
	}

	return &aksCluster.ManagedCluster, nil
}

func GetAKSClusterStatus(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloud *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalClusterStatus, error) {
	cred, err := GetCredentialsForCluster(*cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	aksClient, err := GetAKSClusterClient(cred)
	if err != nil {
		return nil, err
	}
	aksCluster, err := GetAKSCluster(ctx, aksClient, cloud)
	if err != nil {
		return nil, err
	}
	state := apiv2.UNKNOWN
	if aksCluster.Properties != nil {
		var powerState armcontainerservice.Code
		var provisioningState string
		if aksCluster.Properties.PowerState != nil {
			powerState = *aksCluster.Properties.PowerState.Code
		}
		if aksCluster.Properties.ProvisioningState != nil {
			provisioningState = *aksCluster.Properties.ProvisioningState
		}
		state = convertAKSStatus(provisioningState, powerState)
	}

	return &apiv2.ExternalClusterStatus{
		State: state,
	}, nil
}

func convertAKSStatus(provisioningState string, powerState armcontainerservice.Code) apiv2.ExternalClusterState {
	switch {
	case provisioningState == "Creating":
		return apiv2.PROVISIONING
	case provisioningState == "Succeeded" && powerState == "Running":
		return apiv2.RUNNING
	case provisioningState == "Starting":
		return apiv2.PROVISIONING
	case provisioningState == "Stopping":
		return apiv2.STOPPING
	case provisioningState == "Succeeded" && powerState == "Stopped":
		return apiv2.STOPPED
	case provisioningState == "Failed":
		return apiv2.ERROR
	case provisioningState == "Deleting":
		return apiv2.DELETING
	case provisioningState == "Upgrading":
		return apiv2.RECONCILING
	default:
		return apiv2.UNKNOWN
	}
}
