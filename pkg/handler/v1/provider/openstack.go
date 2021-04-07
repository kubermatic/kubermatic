/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

type credentials struct {
	username, password, domain, tenant, tenantID string
}

func getCredentials(userInfo *provider.UserInfo, req OpenstackReq, presetsProvider provider.PresetProvider) (credentials, error) {
	var err error
	var cred credentials
	cred.username, cred.password, cred.domain, cred.tenant, cred.tenantID, err = getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, presetsProvider)
	return cred, fmt.Errorf("error getting OpenStack credentials: %v", err)
}

func OpenstackSizeEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		datacenterName := req.DatacenterName
		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		cred, err := getCredentials(userInfo, req, presetsProvider)
		if err != nil {
			return nil, err
		}

		settings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return providercommon.GetOpenstackSizes(cred.username, cred.password, cred.tenant, cred.tenantID, cred.domain, datacenterName, datacenter, settings.Spec.MachineDeploymentVMResourceQuota)

	}
}

func OpenstackSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackSizeWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, settingsProvider, req.ProjectID, req.ClusterID)
	}
}

func OpenstackTenantEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackTenantReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cred, err := getCredentials(userInfo, req, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.GetOpenstackTenants(userInfo, seedsGetter, cred.username, cred.password, cred.domain, "", "", req.DatacenterName)
	}
}

func OpenstackTenantWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackTenantWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
}

func OpenstackNetworkEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cred, err := getCredentials(userInfo, req, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.GetOpenstackNetworks(userInfo, seedsGetter, cred.username, cred.password, cred.tenant, cred.tenantID, cred.domain, req.DatacenterName)
	}
}

func OpenstackNetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackNetworkWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
}

func OpenstackSecurityGroupEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cred, err := getCredentials(userInfo, req, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.GetOpenstackSecurityGroups(userInfo, seedsGetter, cred.username, cred.password, cred.tenant, cred.tenantID, cred.domain, req.DatacenterName)
	}
}

func OpenstackSecurityGroupWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackSecurityGroupWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
}

func OpenstackSubnetsEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		reqSubnet, ok := request.(OpenstackSubnetReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackSubnetReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		req, _ := request.(OpenstackReq)
		cred, err := getCredentials(userInfo, req, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.GetOpenstackSubnets(userInfo, seedsGetter, cred.username, cred.password, cred.domain, cred.tenant, cred.tenantID, reqSubnet.NetworkID, req.DatacenterName)
	}
}

func OpenstackSubnetsWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackSubnetNoCredentialsReq)
		return providercommon.OpenstackSubnetsWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, req.NetworkID)
	}
}

func OpenstackAvailabilityZoneEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		datacenterName := req.DatacenterName
		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		cred, err := getCredentials(userInfo, req, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.GetOpenstackAvailabilityZones(cred.username, cred.password, cred.tenant, cred.tenantID, cred.domain, datacenterName, datacenter)
	}
}

func OpenstackAvailabilityZoneWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackAvailabilityZoneWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
}

// OpenstackReq represent a request for openstack
// swagger:parameters listOpenstackSizes listOpenstackNetworks listOpenstackSecurityGroups listOpenstackAvailabilityZones
type OpenstackReq struct {
	// in: header
	// Username OpenStack user name
	Username string
	// in: header
	// Password OpenStack user password
	Password string
	// in: header
	// Domain OpenStack domain name
	Domain string
	// in: header
	// Tenant OpenStack tenant name
	Tenant string
	// in: header
	// TenantID OpenStack tenant ID
	TenantID string
	// in: header
	// DatacenterName Openstack datacenter name
	DatacenterName string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
}

func DecodeOpenstackReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.Tenant = r.Header.Get("Tenant")
	req.TenantID = r.Header.Get("TenantID")
	req.Domain = r.Header.Get("Domain")
	req.DatacenterName = r.Header.Get("DatacenterName")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}

// OpenstackNoCredentialsReq represent a request for openstack
// swagger:parameters listOpenstackSizesNoCredentials listOpenstackTenantsNoCredentials listOpenstackNetworksNoCredentials listOpenstackSecurityGroupsNoCredentials listOpenstackAvailabilityZonesNoCredentials
type OpenstackNoCredentialsReq struct {
	common.GetClusterReq
}

func DecodeOpenstackNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackNoCredentialsReq
	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)
	return req, nil
}

// OpenstackSubnetReq represent a request for openstack subnets
// swagger:parameters listOpenstackSubnets
type OpenstackSubnetReq struct {
	OpenstackReq
	// in: query
	NetworkID string `json:"network_id,omitempty"`
}

func DecodeOpenstackSubnetReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackSubnetReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.Domain = r.Header.Get("Domain")
	req.Tenant = r.Header.Get("Tenant")
	req.DatacenterName = r.Header.Get("DatacenterName")
	req.NetworkID = r.URL.Query().Get("network_id")
	if req.NetworkID == "" {
		return nil, fmt.Errorf("get openstack subnets needs a parameter 'network_id'")
	}
	req.Credential = r.Header.Get("Credential")
	return req, nil
}

// OpenstackSubnetNoCredentialsReq represent a request for openstack subnets
// swagger:parameters listOpenstackSubnetsNoCredentials
type OpenstackSubnetNoCredentialsReq struct {
	OpenstackNoCredentialsReq
	// in: query
	NetworkID string `json:"network_id,omitempty"`
}

func DecodeOpenstackSubnetNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackSubnetNoCredentialsReq
	lr, err := DecodeOpenstackNoCredentialsReq(c, r)
	if err != nil {
		return nil, err
	}
	req.OpenstackNoCredentialsReq = lr.(OpenstackNoCredentialsReq)

	req.NetworkID = r.URL.Query().Get("network_id")
	if req.NetworkID == "" {
		return nil, fmt.Errorf("get openstack subnets needs a parameter 'network_id'")
	}

	return req, nil
}

// OpenstackTenantReq represent a request for openstack tenants
// swagger:parameters listOpenstackTenants
type OpenstackTenantReq struct {
	// in: header
	// Username OpenStack user name
	Username string
	// in: header
	// Password OpenStack user password
	Password string
	// in: header
	// Domain OpenStack domain name
	Domain string
	// in: header
	// DatacenterName Openstack datacenter na
	DatacenterName string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
}

func DecodeOpenstackTenantReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackTenantReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.Domain = r.Header.Get("Domain")
	req.DatacenterName = r.Header.Get("DatacenterName")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func getOpenstackCredentials(userInfo *provider.UserInfo, credentialName, username, password, domain, tenant, tenantID string, presetProvider provider.PresetProvider) (string, string, string, string, string, error) {
	if len(credentialName) > 0 {
		preset, err := presetProvider.GetPreset(userInfo, credentialName)
		if err != nil {
			return "", "", "", "", "", fmt.Errorf("can not get preset %s for the user %s", credentialName, userInfo.Email)
		}
		if credentials := preset.Spec.Openstack; credentials != nil {
			username = credentials.Username
			password = credentials.Password
			tenant = credentials.Tenant
			tenantID = credentials.TenantID
			domain = credentials.Domain
		}
	}
	return username, password, domain, tenant, tenantID, nil
}
