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
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"
)

type ClientSet struct {
	Groups           *resources.GroupsClient
	Networks         *network.VirtualNetworksClient
	Subnets          *network.SubnetsClient
	RouteTables      *network.RouteTablesClient
	SecurityGroups   *network.SecurityGroupsClient
	AvailabilitySets *compute.AvailabilitySetsClient
}

// GetClientSet returns a set of clients to interact with various Azure resource types managed by KKP
func GetClientSet(cloud kubermaticv1.CloudSpec, credentials Credentials) (*ClientSet, error) {
	return getClientSet(cloud, credentials)
}

func getClientSet(cloud kubermaticv1.CloudSpec, credentials Credentials) (*ClientSet, error) {

	groupsClient, err := getGroupsClient(cloud, credentials)
	if err != nil {
		return nil, err
	}

	networksClient, err := getNetworksClient(cloud, credentials)
	if err != nil {
		return nil, err
	}

	subnetsClient, err := getSubnetsClient(cloud, credentials)
	if err != nil {
		return nil, err
	}

	routeTablesClient, err := getRouteTablesClient(cloud, credentials)
	if err != nil {
		return nil, err
	}

	securityGroupsClient, err := getSecurityGroupsClient(cloud, credentials)
	if err != nil {
		return nil, err
	}

	availabilitySetsClient, err := getAvailabilitySetClient(cloud, credentials)
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

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
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

func getGroupsClient(cloud kubermaticv1.CloudSpec, credentials Credentials) (*resources.GroupsClient, error) {
	var err error
	groupsClient := resources.NewGroupsClient(credentials.SubscriptionID)
	groupsClient.Authorizer, err = auth.NewClientCredentialsConfig(credentials.ClientID, credentials.ClientSecret, credentials.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &groupsClient, nil
}

func getNetworksClient(cloud kubermaticv1.CloudSpec, credentials Credentials) (*network.VirtualNetworksClient, error) {
	var err error
	networksClient := network.NewVirtualNetworksClient(credentials.SubscriptionID)
	networksClient.Authorizer, err = auth.NewClientCredentialsConfig(credentials.ClientID, credentials.ClientSecret, credentials.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &networksClient, nil
}

func getSubnetsClient(cloud kubermaticv1.CloudSpec, credentials Credentials) (*network.SubnetsClient, error) {
	var err error
	subnetsClient := network.NewSubnetsClient(credentials.SubscriptionID)
	subnetsClient.Authorizer, err = auth.NewClientCredentialsConfig(credentials.ClientID, credentials.ClientSecret, credentials.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &subnetsClient, nil
}

func getRouteTablesClient(cloud kubermaticv1.CloudSpec, credentials Credentials) (*network.RouteTablesClient, error) {
	var err error
	routeTablesClient := network.NewRouteTablesClient(credentials.SubscriptionID)
	routeTablesClient.Authorizer, err = auth.NewClientCredentialsConfig(credentials.ClientID, credentials.ClientSecret, credentials.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &routeTablesClient, nil
}

func getSecurityGroupsClient(cloud kubermaticv1.CloudSpec, credentials Credentials) (*network.SecurityGroupsClient, error) {
	var err error
	securityGroupsClient := network.NewSecurityGroupsClient(credentials.SubscriptionID)
	securityGroupsClient.Authorizer, err = auth.NewClientCredentialsConfig(credentials.ClientID, credentials.ClientSecret, credentials.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &securityGroupsClient, nil
}

func getAvailabilitySetClient(cloud kubermaticv1.CloudSpec, credentials Credentials) (*compute.AvailabilitySetsClient, error) {
	var err error
	asClient := compute.NewAvailabilitySetsClient(credentials.SubscriptionID)
	asClient.Authorizer, err = auth.NewClientCredentialsConfig(credentials.ClientID, credentials.ClientSecret, credentials.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &asClient, nil
}
