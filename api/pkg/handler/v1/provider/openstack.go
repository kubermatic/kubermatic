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
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func OpenstackSizeEndpoint(providers provider.CloudRegistry, datacenters map[string]provider.DatacenterMeta, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}

		datacenterName := req.DatacenterName
		datacenter, found := datacenters[datacenterName]
		if !found {
			return nil, fmt.Errorf("incorrect datacenter name %s", datacenterName)
		}

		username, password, domain, tenant, tenantID := getOpenstackCredentials(req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, credentialManager)

		return getOpenstackSizes(providers, username, password, tenant, tenantID, domain, datacenterName, datacenter)
	}
}

func OpenstackSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry, datacenters map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		openstackSpec := cluster.Spec.Cloud.Openstack
		datacenterName := cluster.Spec.Cloud.DatacenterName

		datacenter, found := datacenters[datacenterName]
		if !found {
			return nil, fmt.Errorf("incorrect datacenter name %s", datacenterName)
		}

		return getOpenstackSizes(providers, openstackSpec.Username, openstackSpec.Password, openstackSpec.Tenant, openstackSpec.TenantID, openstackSpec.Domain, datacenterName, datacenter)
	}
}

func getOpenstackSizes(providers provider.CloudRegistry, username, passowrd, tenant, tenantID, domain, datacenterName string, datacenter provider.DatacenterMeta) ([]apiv1.OpenstackSize, error) {
	osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
	if !ok {
		return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
	}

	osProvider, ok := osProviderInterface.(*openstack.Provider)
	if !ok {
		return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
	}

	flavors, dc, err := osProvider.GetFlavors(kubermaticv1.CloudSpec{
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
			Region:   dc.Spec.Openstack.Region,
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

func OpenstackTenantEndpoint(providers provider.CloudRegistry, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackTenantReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackTenantReq, got = %T", request)
		}

		username, password, domain, _, _ := getOpenstackCredentials(req.Credential, req.Username, req.Password, req.Domain, "", "", credentialManager)

		return getOpenstackTenants(providers, username, password, domain, req.DatacenterName)
	}
}

func OpenstackTenantNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		openstackSpec := cluster.Spec.Cloud.Openstack
		datacenterName := cluster.Spec.Cloud.DatacenterName
		return getOpenstackTenants(providers, openstackSpec.Username, openstackSpec.Password, openstackSpec.Domain, datacenterName)
	}
}

func getOpenstackTenants(providers provider.CloudRegistry, username, password, domain, datacenterName string) ([]apiv1.OpenstackTenant, error) {
	osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
	if !ok {
		return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
	}

	osProvider, ok := osProviderInterface.(*openstack.Provider)
	if !ok {
		return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
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

func OpenstackNetworkEndpoint(providers provider.CloudRegistry, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}

		username, password, domain, tenant, tenantID := getOpenstackCredentials(req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, credentialManager)

		return getOpenstackNetworks(providers, username, password, tenant, tenantID, domain, req.DatacenterName)
	}
}

func OpenstackNetworkNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		openstackSpec := cluster.Spec.Cloud.Openstack
		datacenterName := cluster.Spec.Cloud.DatacenterName
		return getOpenstackNetworks(providers, openstackSpec.Username, openstackSpec.Password, openstackSpec.Tenant, openstackSpec.TenantID, openstackSpec.Domain, datacenterName)
	}
}

func getOpenstackNetworks(providers provider.CloudRegistry, username, password, tenant, tenantID, domain, datacenterName string) ([]apiv1.OpenstackNetwork, error) {
	osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
	if !ok {
		return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
	}

	osProvider, ok := osProviderInterface.(*openstack.Provider)
	if !ok {
		return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
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

func OpenstackSecurityGroupEndpoint(providers provider.CloudRegistry, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}

		username, password, domain, tenant, tenantID := getOpenstackCredentials(req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, credentialManager)

		return getOpenstackSecurityGroups(providers, username, password, tenant, tenantID, domain, req.DatacenterName)
	}
}

func OpenstackSecurityGroupNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		openstackSpec := cluster.Spec.Cloud.Openstack
		datacenterName := cluster.Spec.Cloud.DatacenterName
		return getOpenstackSecurityGroups(providers, openstackSpec.Username, openstackSpec.Password, openstackSpec.Tenant, openstackSpec.TenantID, openstackSpec.Domain, datacenterName)
	}
}

func getOpenstackSecurityGroups(providers provider.CloudRegistry, username, password, tenant, tenantID, domain, datacenterName string) ([]apiv1.OpenstackSecurityGroup, error) {
	osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
	if !ok {
		return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
	}

	osProvider, ok := osProviderInterface.(*openstack.Provider)
	if !ok {
		return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
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

func OpenstackSubnetsEndpoint(providers provider.CloudRegistry, credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackSubnetReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackSubnetReq, got = %T", request)
		}

		username, password, domain, tenant, tenantID := getOpenstackCredentials(req.Credential, req.Username, req.Password, req.Domain, req.Tenant, req.TenantID, credentialManager)

		return getOpenstackSubnets(providers, username, password, domain, tenant, tenantID, req.NetworkID, req.DatacenterName)
	}
}

func OpenstackSubnetsNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackSubnetNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		openstackSpec := cluster.Spec.Cloud.Openstack
		datacenterName := cluster.Spec.Cloud.DatacenterName
		return getOpenstackSubnets(providers, openstackSpec.Username, openstackSpec.Password, openstackSpec.Domain, openstackSpec.Tenant, openstackSpec.TenantID, req.NetworkID, datacenterName)
	}
}

func getOpenstackSubnets(providers provider.CloudRegistry, username, password, domain, tenant, tenantID, networkID, datacenterName string) ([]apiv1.OpenstackSubnet, error) {
	osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
	if !ok {
		return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
	}

	osProvider, ok := osProviderInterface.(*openstack.Provider)
	if !ok {
		return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
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

func getOpenstackCredentials(credentialName, username, password, domain, tenant, tenantID string, credentialManager common.PresetsManager) (string, string, string, string, string) {
	if len(credentialName) > 0 && credentialManager.GetPresets().Openstack.Credentials != nil {
		for _, credential := range credentialManager.GetPresets().Openstack.Credentials {
			if credential.Name == credentialName {
				username = credential.Username
				password = credential.Password
				tenant = credential.Tenant
				tenantID = credential.TenantID
				domain = credential.Domain
				break
			}
		}
	}

	return username, password, domain, tenant, tenantID
}
