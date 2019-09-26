package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func OpenstackSizeEndpoint(seedsGetter provider.SeedsGetter, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, ok := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "can not get user info")
		}

		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}

		datacenterName := req.DatacenterName
		_, datacenter, err := provider.DatacenterFromSeedMap(seeds, datacenterName)
		if err != nil {
			return nil, fmt.Errorf("error getting dc: %v", err)
		}

		username, password, domain, tenant, tenantID, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, credentialManager)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackSizes(username, password, tenant, tenantID, domain, datacenterName, datacenter)
	}
}

func OpenstackSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}

		_, datacenter, err := provider.DatacenterFromSeedMap(seeds, datacenterName)
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

func getOpenstackSizes(username, passowrd, tenant, tenantID, domain, datacenterName string, datacenter *kubermaticv1.Datacenter) ([]apiv1.OpenstackSize, error) {

	provider, err := openstack.NewCloudProvider(datacenter, nil)
	if err != nil {
		return nil, err
	}
	flavors, err := provider.GetFlavors(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			Username: username,
			Password: passowrd,
			Tenant:   tenant,
			TenantID: tenantID,
			Domain:   domain,
		},
	})
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

func OpenstackTenantEndpoint(seedsGetter provider.SeedsGetter, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackTenantReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackTenantReq, got = %T", request)
		}
		userInfo, ok := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "can not get user info")
		}
		username, password, domain, _, _, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, "", "", credentialManager)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackTenants(seedsGetter, username, password, domain, req.DatacenterName)
	}
}

func OpenstackTenantWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

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
		return getOpenstackTenants(seedsGetter, creds.Username, creds.Password, creds.Domain, datacenterName)
	}
}

func getOpenstackTenants(seedsGetter provider.SeedsGetter, username, password, domain, datacenterName string) ([]apiv1.OpenstackTenant, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}

	osProvider, err := getOpenstackCloudProvider(seeds, datacenterName)
	if err != nil {
		return nil, err
	}

	tenants, err := osProvider.GetTenants(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			Username: username,
			Password: password,
			Domain:   domain,
		},
	})
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

func OpenstackNetworkEndpoint(seedsGetter provider.SeedsGetter, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, ok := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "can not get user info")
		}
		username, password, domain, tenant, tenantID, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, credentialManager)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackNetworks(seedsGetter, username, password, tenant, tenantID, domain, req.DatacenterName)
	}
}

func OpenstackNetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

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
		return getOpenstackNetworks(seedsGetter, creds.Username, creds.Password, creds.Tenant, creds.TenantID, creds.Domain, datacenterName)
	}
}

func getOpenstackNetworks(seedsGetter provider.SeedsGetter, username, password, tenant, tenantID, domain, datacenterName string) ([]apiv1.OpenstackNetwork, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	osProvider, err := getOpenstackCloudProvider(seeds, datacenterName)
	if err != nil {
		return nil, err
	}

	networks, err := osProvider.GetNetworks(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			Username: username,
			Password: password,
			Tenant:   tenant,
			TenantID: tenantID,
			Domain:   domain,
		},
	})
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

func OpenstackSecurityGroupEndpoint(seedsGetter provider.SeedsGetter, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		userInfo, ok := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "can not get user info")
		}
		username, password, domain, tenant, tenantID, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, credentialManager)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackSecurityGroups(seedsGetter, username, password, tenant, tenantID, domain, req.DatacenterName)
	}
}

func OpenstackSecurityGroupWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

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
		return getOpenstackSecurityGroups(seedsGetter, creds.Username, creds.Password, creds.Tenant, creds.TenantID, creds.Domain, datacenterName)
	}
}

func getOpenstackSecurityGroups(seedsGetter provider.SeedsGetter, username, password, tenant, tenantID, domain, datacenterName string) ([]apiv1.OpenstackSecurityGroup, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	osProvider, err := getOpenstackCloudProvider(seeds, datacenterName)
	if err != nil {
		return nil, err
	}

	securityGroups, err := osProvider.GetSecurityGroups(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			Username: username,
			Password: password,
			Tenant:   tenant,
			TenantID: tenantID,
			Domain:   domain,
		},
	})
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

func OpenstackSubnetsEndpoint(seedsGetter provider.SeedsGetter, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackSubnetReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackSubnetReq, got = %T", request)
		}
		userInfo, ok := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "can not get user info")
		}
		username, password, domain, tenant, tenantID, err := getOpenstackCredentials(userInfo, req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, credentialManager)
		if err != nil {
			return nil, fmt.Errorf("error getting OpenStack credentials: %v", err)
		}
		return getOpenstackSubnets(seedsGetter, username, password, domain, tenant, tenantID, req.NetworkID, req.DatacenterName)
	}
}

func OpenstackSubnetsWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackSubnetNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

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
		return getOpenstackSubnets(seedsGetter, creds.Username, creds.Password, creds.Domain, creds.Tenant, creds.TenantID, req.NetworkID, datacenterName)
	}
}

func getOpenstackSubnets(seedsGetter provider.SeedsGetter, username, password, domain, tenant, tenantID, networkID, datacenterName string) ([]apiv1.OpenstackSubnet, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	osProvider, err := getOpenstackCloudProvider(seeds, datacenterName)
	if err != nil {
		return nil, err
	}

	subnets, err := osProvider.GetSubnets(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			Username: username,
			Password: password,
			Domain:   domain,
			Tenant:   tenant,
			TenantID: tenantID,
		},
	}, networkID)
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

func getClusterForOpenstack(ctx context.Context, projectProvider provider.ProjectProvider, projectID string, clusterID string) (*kubermaticv1.Cluster, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
	_, err := projectProvider.Get(userInfo, projectID, &provider.ProjectGetOptions{})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	cluster, err := clusterProvider.Get(userInfo, clusterID, &provider.ClusterGetOptions{})
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	if cluster.Spec.Cloud.Openstack == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}
	return cluster, nil
}

// OpenstackReq represent a request for openstack
type OpenstackReq struct {
	Username       string
	Password       string
	Domain         string
	Tenant         string
	TenantID       string
	DatacenterName string
	Credential     string
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
// swagger:parameters listOpenstackSizesNoCredentials listOpenstackTenantsNoCredentials listOpenstackNetworksNoCredentials listOpenstackSecurityGroupsNoCredentials
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
	NetworkID string
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
	NetworkID string
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
type OpenstackTenantReq struct {
	Username       string
	Password       string
	Domain         string
	DatacenterName string
	Credential     string
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

func getOpenstackCredentials(userInfo *provider.UserInfo, credentialName, username, password, domain, tenant, tenantID string, credentialManager common.PresetsManager) (string, string, string, string, string, error) {
	if len(credentialName) > 0 {
		preset, err := credentialManager.GetPreset(userInfo, credentialName)
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

func getOpenstackCloudProvider(seeds map[string]*kubermaticv1.Seed, datacenterName string) (*openstack.Provider, error) {
	_, dc, err := provider.DatacenterFromSeedMap(seeds, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find datacenter %q: %v", datacenterName, err)
	}
	osProvider, err := openstack.NewCloudProvider(dc, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Openstack provider: %v", err)
	}

	return osProvider, nil
}
