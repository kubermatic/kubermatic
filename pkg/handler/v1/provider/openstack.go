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
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

type credentials struct {
	username, password, tenant, tenantID, domain string
}

func getAuthInfo(ctx context.Context, req OpenstackReq, userInfoGetter provider.UserInfoGetter, presetsProvider provider.PresetProvider) (*provider.UserInfo, *credentials, error) {
	var cred *credentials
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, nil, common.KubernetesErrorToHTTPError(err)
	}
	// No preset is used
	presetName := req.Credential
	if presetName == "" {
		return userInfo, &credentials{
			username: req.Username,
			password: req.Password,
			domain:   req.Domain,
			tenant:   req.Tenant,
			tenantID: req.TenantID,
		}, nil
	}
	// Preset is used
	cred, err = getPresetCredentials(userInfo, presetName, presetsProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting preset credentials for OpenStack: %v", err)
	}
	return userInfo, cred, nil
}

func OpenstackSizeEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider,
	userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, cred, err := getAuthInfo(ctx, req, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		datacenterName := req.DatacenterName
		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}
		settings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return providercommon.GetOpenstackSizes(cred.username, cred.password, cred.tenant, cred.tenantID, cred.domain,
			datacenterName, datacenter, settings.Spec.MachineDeploymentVMResourceQuota, caBundle)
	}
}

func OpenstackSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackSizeWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider,
			privilegedProjectProvider, seedsGetter, settingsProvider, req.ProjectID, req.ClusterID, caBundle)
	}
}

func OpenstackTenantEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		reqTenant, ok := request.(OpenstackTenantReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackTenantReq, got = %T", request)
		}
		req := OpenstackReq{
			reqTenant.Username, reqTenant.Password, reqTenant.Domain, "", "", reqTenant.DatacenterName, reqTenant.Credential,
		}
		userInfo, cred, err := getAuthInfo(ctx, req, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.GetOpenstackTenants(userInfo, seedsGetter, cred.username, cred.password, cred.domain,
			"", "", reqTenant.DatacenterName, caBundle)
	}
}

func OpenstackTenantWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackTenantWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider,
			privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, caBundle)
	}
}

func OpenstackNetworkEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, cred, err := getAuthInfo(ctx, req, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.GetOpenstackNetworks(userInfo, seedsGetter, cred.username, cred.password, cred.tenant,
			cred.tenantID, cred.domain, req.DatacenterName, caBundle)
	}
}

func OpenstackNetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackNetworkWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider,
			privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, caBundle)
	}
}

func OpenstackSecurityGroupEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, cred, err := getAuthInfo(ctx, req, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.GetOpenstackSecurityGroups(userInfo, seedsGetter, cred.username, cred.password, cred.tenant,
			cred.tenantID, cred.domain, req.DatacenterName, caBundle)
	}
}

func OpenstackSecurityGroupWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackSecurityGroupWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider,
			privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, caBundle)
	}
}

func OpenstackSubnetsEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackSubnetReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackSubnetReq, got = %T", request)
		}
		userInfo, cred, err := getAuthInfo(ctx, req.OpenstackReq, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.GetOpenstackSubnets(userInfo, seedsGetter, cred.username, cred.password, cred.domain, cred.tenant,
			cred.tenantID, req.NetworkID, req.DatacenterName, caBundle)
	}
}

func OpenstackSubnetsWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackSubnetNoCredentialsReq)
		return providercommon.OpenstackSubnetsWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider,
			privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, req.NetworkID, caBundle)
	}
}

func OpenstackAvailabilityZoneEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, cred, err := getAuthInfo(ctx, req, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		datacenterName := req.DatacenterName
		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}
		return providercommon.GetOpenstackAvailabilityZones(cred.username, cred.password, cred.tenant, cred.tenantID, cred.domain,
			datacenterName, datacenter, caBundle)
	}
}

func OpenstackAvailabilityZoneWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	userInfoGetter provider.UserInfoGetter, caBundle *x509.CertPool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		return providercommon.OpenstackAvailabilityZoneWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider,
			privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, caBundle)
	}
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
	// DatacenterName Openstack datacenter name
	DatacenterName string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
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

func DecodeOpenstackTenantReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackTenantReq
	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.Domain = r.Header.Get("Domain")
	req.DatacenterName = r.Header.Get("DatacenterName")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}

func getPresetCredentials(userInfo *provider.UserInfo, presetName string, presetProvider provider.PresetProvider) (*credentials, error) {
	p, err := presetProvider.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, fmt.Errorf("can not get preset %s for the user %s", presetName, userInfo.Email)
	}
	if p.Spec.Openstack == nil {
		return nil, fmt.Errorf("credentials for OpenStack provider not present in preset %s for the user %s", presetName, userInfo.Email)
	}
	return &credentials{
		username: p.Spec.Openstack.Username,
		password: p.Spec.Openstack.Password,
		domain:   p.Spec.Openstack.Domain,
		tenant:   p.Spec.Openstack.Tenant,
		tenantID: p.Spec.Openstack.TenantID,
	}, nil
}
