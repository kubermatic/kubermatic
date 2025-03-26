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

package azure

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"
)

// ResourceGroupClient is the subset of functions we need from armresources.VirtualResourceGroupsClient;
// this interface is purely here for allowing unit tests.
type ResourceGroupClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, parameters armresources.ResourceGroup, options *armresources.ResourceGroupsClientCreateOrUpdateOptions) (armresources.ResourceGroupsClientCreateOrUpdateResponse, error)
	Get(ctx context.Context, resourceGroupName string, options *armresources.ResourceGroupsClientGetOptions) (armresources.ResourceGroupsClientGetResponse, error)
	BeginDelete(ctx context.Context, resourceGroupName string, options *armresources.ResourceGroupsClientBeginDeleteOptions) (*runtime.Poller[armresources.ResourceGroupsClientDeleteResponse], error)
}

// NetworkClient is the subset of functions we need from armnetwork.VirtualNetworksClient;
// this interface is purely here for allowing unit tests.
type NetworkClient interface {
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, virtualNetworkName string, parameters armnetwork.VirtualNetwork, options *armnetwork.VirtualNetworksClientBeginCreateOrUpdateOptions) (*runtime.Poller[armnetwork.VirtualNetworksClientCreateOrUpdateResponse], error)
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientGetOptions) (armnetwork.VirtualNetworksClientGetResponse, error)
	BeginDelete(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientBeginDeleteOptions) (*runtime.Poller[armnetwork.VirtualNetworksClientDeleteResponse], error)
}

// SubnetClient is the subset of functions we need from armnetwork.SubnetsClient;
// this interface is purely here for allowing unit tests.
type SubnetClient interface {
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, subnetParameters armnetwork.Subnet, options *armnetwork.SubnetsClientBeginCreateOrUpdateOptions) (*runtime.Poller[armnetwork.SubnetsClientCreateOrUpdateResponse], error)
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, options *armnetwork.SubnetsClientGetOptions) (armnetwork.SubnetsClientGetResponse, error)
	BeginDelete(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, options *armnetwork.SubnetsClientBeginDeleteOptions) (*runtime.Poller[armnetwork.SubnetsClientDeleteResponse], error)
}

// RouteTableClient is the subset of functions we need from armnetwork.RouteTablesClient;
// this interface is purely here for allowing unit tests.
type RouteTableClient interface {
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, routeTableName string, parameters armnetwork.RouteTable, options *armnetwork.RouteTablesClientBeginCreateOrUpdateOptions) (*runtime.Poller[armnetwork.RouteTablesClientCreateOrUpdateResponse], error)
	Get(ctx context.Context, resourceGroupName string, routeTableName string, options *armnetwork.RouteTablesClientGetOptions) (armnetwork.RouteTablesClientGetResponse, error)
	BeginDelete(ctx context.Context, resourceGroupName string, routeTableName string, options *armnetwork.RouteTablesClientBeginDeleteOptions) (*runtime.Poller[armnetwork.RouteTablesClientDeleteResponse], error)
}

// SecurityGroupClient is the subset of functions we need from armnetwork.SecurityGroupsClient;
// this interface is purely here for allowing unit tests.
type SecurityGroupClient interface {
	BeginCreateOrUpdate(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, parameters armnetwork.SecurityGroup, options *armnetwork.SecurityGroupsClientBeginCreateOrUpdateOptions) (*runtime.Poller[armnetwork.SecurityGroupsClientCreateOrUpdateResponse], error)
	Get(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, options *armnetwork.SecurityGroupsClientGetOptions) (armnetwork.SecurityGroupsClientGetResponse, error)
	BeginDelete(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, options *armnetwork.SecurityGroupsClientBeginDeleteOptions) (*runtime.Poller[armnetwork.SecurityGroupsClientDeleteResponse], error)
}

// AvailabilitySetClient is the subset of functions we need from armcompute.AvailabilitySetsClient;
// this interface is purely here for allowing unit tests.
type AvailabilitySetClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, availabilitySetName string, parameters armcompute.AvailabilitySet, options *armcompute.AvailabilitySetsClientCreateOrUpdateOptions) (armcompute.AvailabilitySetsClientCreateOrUpdateResponse, error)
	Get(ctx context.Context, resourceGroupName string, availabilitySetName string, options *armcompute.AvailabilitySetsClientGetOptions) (armcompute.AvailabilitySetsClientGetResponse, error)
	Delete(ctx context.Context, resourceGroupName string, availabilitySetName string, options *armcompute.AvailabilitySetsClientDeleteOptions) (armcompute.AvailabilitySetsClientDeleteResponse, error)
}

// ClientSet provides a set of Azure service clients that are necessary to reconcile resources needed by KKP.
type ClientSet struct {
	Groups           ResourceGroupClient
	Networks         NetworkClient
	Subnets          SubnetClient
	RouteTables      RouteTableClient
	SecurityGroups   SecurityGroupClient
	AvailabilitySets AvailabilitySetClient
}

// GetClientSet returns a ClientSet using the passed credentials as authorization.
func GetClientSet(credentials Credentials) (*ClientSet, error) {
	credential, err := credentials.ToAzureCredential()
	if err != nil {
		return nil, err
	}

	groupsClient, err := getGroupsClient(credential, credentials.SubscriptionID)
	if err != nil {
		return nil, err
	}

	networksClient, err := getNetworksClient(credential, credentials.SubscriptionID)
	if err != nil {
		return nil, err
	}

	subnetsClient, err := getSubnetsClient(credential, credentials.SubscriptionID)
	if err != nil {
		return nil, err
	}

	routeTablesClient, err := getRouteTablesClient(credential, credentials.SubscriptionID)
	if err != nil {
		return nil, err
	}

	securityGroupsClient, err := getSecurityGroupsClient(credential, credentials.SubscriptionID)
	if err != nil {
		return nil, err
	}

	availabilitySetsClient, err := getAvailabilitySetClient(credential, credentials.SubscriptionID)
	if err != nil {
		return nil, err
	}

	return &ClientSet{
		Groups:           groupsClient,
		Networks:         networksClient,
		Subnets:          subnetsClient,
		RouteTables:      routeTablesClient,
		SecurityGroups:   securityGroupsClient,
		AvailabilitySets: availabilitySetsClient,
	}, nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (Credentials, error) {
	tenantID := cloud.Azure.TenantID
	subscriptionID := cloud.Azure.SubscriptionID
	clientID := cloud.Azure.ClientID
	clientSecret := cloud.Azure.ClientSecret
	var err error

	if tenantID == "" {
		if cloud.Azure.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		tenantID, err = secretKeySelector(cloud.Azure.CredentialsReference, kubermaticresources.AzureTenantID)
		if err != nil {
			return Credentials{}, err
		}
	}

	if subscriptionID == "" {
		if cloud.Azure.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		subscriptionID, err = secretKeySelector(cloud.Azure.CredentialsReference, kubermaticresources.AzureSubscriptionID)
		if err != nil {
			return Credentials{}, err
		}
	}

	if clientID == "" {
		if cloud.Azure.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		clientID, err = secretKeySelector(cloud.Azure.CredentialsReference, kubermaticresources.AzureClientID)
		if err != nil {
			return Credentials{}, err
		}
	}

	if clientSecret == "" {
		if cloud.Azure.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		clientSecret, err = secretKeySelector(cloud.Azure.CredentialsReference, kubermaticresources.AzureClientSecret)
		if err != nil {
			return Credentials{}, err
		}
	}

	return Credentials{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}, nil
}

func ValidateCredentials(ctx context.Context, credentials *azidentity.ClientSecretCredential, subscriptionID string) error {
	subscriptionClient, err := armsubscription.NewSubscriptionsClient(credentials, nil)
	if err != nil {
		return err
	}

	_, err = subscriptionClient.Get(ctx, subscriptionID, nil)

	return err
}

func getGroupsClient(credentials *azidentity.ClientSecretCredential, subscriptionID string) (*armresources.ResourceGroupsClient, error) {
	return armresources.NewResourceGroupsClient(subscriptionID, credentials, nil)
}

func getNetworksClient(credentials *azidentity.ClientSecretCredential, subscriptionID string) (*armnetwork.VirtualNetworksClient, error) {
	return armnetwork.NewVirtualNetworksClient(subscriptionID, credentials, nil)
}

func getSubnetsClient(credentials *azidentity.ClientSecretCredential, subscriptionID string) (*armnetwork.SubnetsClient, error) {
	return armnetwork.NewSubnetsClient(subscriptionID, credentials, nil)
}

func getRouteTablesClient(credentials *azidentity.ClientSecretCredential, subscriptionID string) (*armnetwork.RouteTablesClient, error) {
	return armnetwork.NewRouteTablesClient(subscriptionID, credentials, nil)
}

func getSecurityGroupsClient(credentials *azidentity.ClientSecretCredential, subscriptionID string) (*armnetwork.SecurityGroupsClient, error) {
	return armnetwork.NewSecurityGroupsClient(subscriptionID, credentials, nil)
}

func getAvailabilitySetClient(credentials *azidentity.ClientSecretCredential, subscriptionID string) (*armcompute.AvailabilitySetsClient, error) {
	return armcompute.NewAvailabilitySetsClient(subscriptionID, credentials, nil)
}

func getSizesClient(credentials *azidentity.ClientSecretCredential, subscriptionID string) (*armcompute.VirtualMachineSizesClient, error) {
	return armcompute.NewVirtualMachineSizesClient(subscriptionID, credentials, nil)
}
