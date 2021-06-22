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
	"strings"

	"github.com/go-kit/kit/endpoint"

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func getAuthInfo(ctx context.Context, req OpenstackReq, userInfoGetter provider.UserInfoGetter, presetsProvider provider.PresetProvider) (*provider.UserInfo, *resources.OpenstackCredentials, error) {
	var cred *resources.OpenstackCredentials
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, nil, common.KubernetesErrorToHTTPError(err)
	}

	t := ctx.Value(middleware.RawTokenContextKey)
	token, ok := t.(string)
	if !ok || token == "" {
		return nil, nil, k8cerrors.NewNotAuthorized()
	}

	// No preset is used
	presetName := req.Credential
	if presetName == "" {
		credentials := &resources.OpenstackCredentials{
			Username:                    req.Username,
			Password:                    req.Password,
			Tenant:                      req.Tenant,
			TenantID:                    req.TenantID,
			Domain:                      req.Domain,
			ApplicationCredentialID:     req.ApplicationCredentialID,
			ApplicationCredentialSecret: req.ApplicationCredentialSecret,
		}
		if req.OIDCAuthentication {
			credentials.Token = token
		}
		return userInfo, credentials, nil
	}
	// Preset is used
	cred, err = getPresetCredentials(userInfo, presetName, presetsProvider, token)
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
		return providercommon.GetOpenstackSizes(cred, datacenter, settings.Spec.MachineDeploymentVMResourceQuota, caBundle)
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
			Username:                    reqTenant.Username,
			Password:                    reqTenant.Password,
			Domain:                      reqTenant.Domain,
			Tenant:                      "",
			TenantID:                    "",
			DatacenterName:              reqTenant.DatacenterName,
			ApplicationCredentialID:     reqTenant.ApplicationCredentialID,
			ApplicationCredentialSecret: reqTenant.ApplicationCredentialSecret,
			Credential:                  reqTenant.Credential,
			OIDCAuthentication:          reqTenant.OIDCAuthentication,
		}

		userInfo, cred, err := getAuthInfo(ctx, req, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.GetOpenstackTenants(userInfo, seedsGetter, cred, reqTenant.DatacenterName, caBundle)
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
		return providercommon.GetOpenstackNetworks(userInfo, seedsGetter, cred, req.DatacenterName, caBundle)
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
		return providercommon.GetOpenstackSecurityGroups(userInfo, seedsGetter, cred, req.DatacenterName, caBundle)
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
		return providercommon.GetOpenstackSubnets(userInfo, seedsGetter, cred, req.NetworkID, req.DatacenterName, caBundle)
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
		return providercommon.GetOpenstackAvailabilityZones(datacenter, cred, caBundle)
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
	req.ApplicationCredentialID = r.Header.Get("ApplicationCredentialID")
	req.ApplicationCredentialSecret = r.Header.Get("ApplicationCredentialSecret")
	req.OIDCAuthentication = strings.EqualFold(r.Header.Get("OIDCAuthentication"), "true")
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
	req.ApplicationCredentialID = r.Header.Get("ApplicationCredentialID")
	req.ApplicationCredentialSecret = r.Header.Get("ApplicationCredentialSecret")
	req.OIDCAuthentication = strings.EqualFold(r.Header.Get("OIDCAuthentication"), "true")

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
	// ApplicationCredentialID application credential ID
	ApplicationCredentialID string
	// in: header
	// ApplicationCredentialSecret application credential Secret
	ApplicationCredentialSecret string
	// in: header
	// OIDCAuthentication when true use OIDC token
	OIDCAuthentication bool
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
	// ApplicationCredentialID application credential ID
	ApplicationCredentialID string
	// in: header
	// ApplicationCredentialSecret application credential Secret
	ApplicationCredentialSecret string
	// in: header
	// OIDCAuthentication when true use OIDC token
	OIDCAuthentication bool

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
	req.ApplicationCredentialID = r.Header.Get("ApplicationCredentialID")
	req.ApplicationCredentialSecret = r.Header.Get("ApplicationCredentialSecret")
	req.OIDCAuthentication = strings.EqualFold(r.Header.Get("OIDCAuthentication"), "true")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}

func getPresetCredentials(userInfo *provider.UserInfo, presetName string, presetProvider provider.PresetProvider, token string) (*resources.OpenstackCredentials, error) {
	p, err := presetProvider.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, fmt.Errorf("can not get preset %s for the user %s", presetName, userInfo.Email)
	}
	if p.Spec.Openstack == nil {
		return nil, fmt.Errorf("credentials for OpenStack provider not present in preset %s for the user %s", presetName, userInfo.Email)
	}
	credentials := &resources.OpenstackCredentials{
		Username:                    p.Spec.Openstack.Username,
		Password:                    p.Spec.Openstack.Password,
		Tenant:                      p.Spec.Openstack.Tenant,
		TenantID:                    p.Spec.Openstack.TenantID,
		Domain:                      p.Spec.Openstack.Domain,
		ApplicationCredentialID:     p.Spec.Openstack.ApplicationCredentialID,
		ApplicationCredentialSecret: p.Spec.Openstack.ApplicationCredentialSecret,
	}

	if p.Spec.Openstack.UseToken {
		credentials.Token = token
	}

	return credentials, nil
}
