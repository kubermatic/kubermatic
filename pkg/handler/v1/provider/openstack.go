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

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func OpenstackSizeEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
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

		username, password, domain, tenant, tenantID, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, presetsProvider)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackSizes(username, password, tenant, tenantID, domain, datacenterName, datacenter)
	}
}

func OpenstackSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		creds, err := openstack.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}

		return getOpenstackSizes(creds.Username, creds.Password, creds.Tenant, creds.TenantID, creds.Domain, datacenterName, datacenter)
	}
}

func getOpenstackSizes(username, password, tenant, tenantID, domain, datacenterName string, datacenter *kubermaticv1.Datacenter) ([]apiv1.OpenstackSize, error) {
	flavors, err := openstack.GetFlavors(username, password, domain, tenant, tenantID, datacenter.Spec.Openstack.AuthURL, datacenter.Spec.Openstack.Region)
	if err != nil {
		return nil, err
	}

	apiSizes := []apiv1.OpenstackSize{}
	for _, flavor := range flavors {
		apiSize := apiv1.OpenstackSize{
			Slug:     flavor.Name,
			Memory:   flavor.RAM,
			VCPUs:    flavor.VCPUs,
			Disk:     flavor.Disk,
			Swap:     flavor.Swap,
			Region:   datacenter.Spec.Openstack.Region,
			IsPublic: flavor.IsPublic,
		}
		if MeetsOpenstackNodeSizeRequirement(apiSize, datacenter.Spec.Openstack.NodeSizeRequirements) {
			apiSizes = append(apiSizes, apiSize)
		}
	}

	return apiSizes, nil
}

func MeetsOpenstackNodeSizeRequirement(apiSize apiv1.OpenstackSize, requirements kubermaticv1.OpenstackNodeSizeRequirements) bool {
	if apiSize.VCPUs < requirements.MinimumVCPUs {
		return false
	}
	if apiSize.Memory < requirements.MinimumMemory {
		return false
	}
	return true
}

func OpenstackTenantEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackTenantReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackTenantReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		username, password, domain, _, _, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, "", "", presetsProvider)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackTenants(userInfo, seedsGetter, username, password, domain, "", "", req.DatacenterName)
	}
}

func OpenstackTenantWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		creds, err := openstack.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}
		return getOpenstackTenants(userInfo, seedsGetter, creds.Username, creds.Password, creds.Domain, creds.Tenant, creds.TenantID, datacenterName)
	}
}

func getOpenstackTenants(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, username, password, domain, tenant, tenantID, datacenterName string) ([]apiv1.OpenstackTenant, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	tenants, err := openstack.GetTenants(username, password, domain, tenant, tenantID, authURL, region)
	if err != nil {
		return nil, fmt.Errorf("couldn't get tenants: %v", err)
	}

	apiTenants := []apiv1.OpenstackTenant{}
	for _, tenant := range tenants {
		apiTenant := apiv1.OpenstackTenant{
			Name: tenant.Name,
			ID:   tenant.ID,
		}

		apiTenants = append(apiTenants, apiTenant)
	}

	return apiTenants, nil
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
		username, password, domain, tenant, tenantID, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, presetsProvider)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackNetworks(userInfo, seedsGetter, username, password, tenant, tenantID, domain, req.DatacenterName)
	}
}

func OpenstackNetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		creds, err := openstack.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}
		return getOpenstackNetworks(userInfo, seedsGetter, creds.Username, creds.Password, creds.Tenant, creds.TenantID, creds.Domain, datacenterName)
	}
}

func getOpenstackNetworks(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, username, password, tenant, tenantID, domain, datacenterName string) ([]apiv1.OpenstackNetwork, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	networks, err := openstack.GetNetworks(username, password, domain, tenant, tenantID, authURL, region)
	if err != nil {
		return nil, err
	}

	apiNetworks := []apiv1.OpenstackNetwork{}
	for _, network := range networks {
		apiNetwork := apiv1.OpenstackNetwork{
			Name:     network.Name,
			ID:       network.ID,
			External: network.External,
		}

		apiNetworks = append(apiNetworks, apiNetwork)
	}

	return apiNetworks, nil
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
		username, password, domain, tenant, tenantID, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, presetsProvider)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackSecurityGroups(userInfo, seedsGetter, username, password, tenant, tenantID, domain, req.DatacenterName)
	}
}

func OpenstackSecurityGroupWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		userInfo, ok := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "can not get user info")
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		creds, err := openstack.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}
		return getOpenstackSecurityGroups(userInfo, seedsGetter, creds.Username, creds.Password, creds.Tenant, creds.TenantID, creds.Domain, datacenterName)
	}
}

func getOpenstackSecurityGroups(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, username, password, tenant, tenantID, domain, datacenterName string) ([]apiv1.OpenstackSecurityGroup, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	securityGroups, err := openstack.GetSecurityGroups(username, password, domain, tenant, tenantID, authURL, region)
	if err != nil {
		return nil, err
	}

	apiSecurityGroups := []apiv1.OpenstackSecurityGroup{}
	for _, securityGroup := range securityGroups {
		apiSecurityGroup := apiv1.OpenstackSecurityGroup{
			Name: securityGroup.Name,
			ID:   securityGroup.ID,
		}

		apiSecurityGroups = append(apiSecurityGroups, apiSecurityGroup)
	}

	return apiSecurityGroups, nil
}

func OpenstackSubnetsEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackSubnetReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackSubnetReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		username, password, domain, tenant, tenantID, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, presetsProvider)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackSubnets(userInfo, seedsGetter, username, password, domain, tenant, tenantID, req.NetworkID, req.DatacenterName)
	}
}

func OpenstackSubnetsWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackSubnetNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		creds, err := openstack.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}
		return getOpenstackSubnets(userInfo, seedsGetter, creds.Username, creds.Password, creds.Domain, creds.Tenant, creds.TenantID, req.NetworkID, datacenterName)
	}
}

func getOpenstackSubnets(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, username, password, domain, tenant, tenantID, networkID, datacenterName string) ([]apiv1.OpenstackSubnet, error) {
	authURL, region, err := getOpenstackAuthURLAndRegion(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, err
	}

	subnets, err := openstack.GetSubnets(username, password, domain, tenant, tenantID, networkID, authURL, region)
	if err != nil {
		return nil, err
	}

	apiSubnetIDs := []apiv1.OpenstackSubnet{}
	for _, subnet := range subnets {
		apiSubnetIDs = append(apiSubnetIDs, apiv1.OpenstackSubnet{
			ID:   subnet.ID,
			Name: subnet.Name,
		})
	}

	return apiSubnetIDs, nil
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

		username, password, domain, tenant, tenantID, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, presetsProvider)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackAvailabilityZones(username, password, tenant, tenantID, domain, datacenterName, datacenter)
	}
}

func OpenstackAvailabilityZoneWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		creds, err := openstack.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}

		return getOpenstackAvailabilityZones(creds.Username, creds.Password, creds.Tenant, creds.TenantID, creds.Domain, datacenterName, datacenter)
	}
}

func getOpenstackAvailabilityZones(username, password, tenant, tenantID, domain, datacenterName string, datacenter *kubermaticv1.Datacenter) ([]apiv1.OpenstackAvailabilityZone, error) {
	availabilityZones, err := openstack.GetAvailabilityZones(username, password, domain, tenant, tenantID, datacenter.Spec.Openstack.AuthURL, datacenter.Spec.Openstack.Region)
	if err != nil {
		return nil, err
	}

	apiAvailabilityZones := []apiv1.OpenstackAvailabilityZone{}
	for _, availabilityZone := range availabilityZones {
		apiAvailabilityZones = append(apiAvailabilityZones, apiv1.OpenstackAvailabilityZone{
			Name: availabilityZone.ZoneName,
		})
	}

	return apiAvailabilityZones, nil
}

func getClusterForOpenstack(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, projectID string, clusterID string) (*kubermaticv1.Cluster, error) {
	cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Openstack == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}
	return cluster, nil
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

func DecodeOpenstackReq(c context.Context, r *http.Request) (interface{}, error) {
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

func DecodeOpenstackSubnetReq(c context.Context, r *http.Request) (interface{}, error) {
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

func DecodeOpenstackTenantReq(c context.Context, r *http.Request) (interface{}, error) {
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

func getOpenstackAuthURLAndRegion(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, datacenterName string) (string, string, error) {
	_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return "", "", fmt.Errorf("failed to find datacenter %q: %v", datacenterName, err)
	}
	return dc.Spec.Openstack.AuthURL, dc.Spec.Openstack.Region, nil
}
