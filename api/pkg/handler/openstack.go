package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func openstackSizeEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}

		return getOpenstackSizes(providers, req.Username, req.Password, req.Tenant, req.Domain, req.DatacenterName)
	}
}

func openstackSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		openstackSpec := cluster.Spec.Cloud.Openstack
		datacenterName := cluster.Spec.Cloud.DatacenterName
		return getOpenstackSizes(providers, openstackSpec.Username, openstackSpec.Password, openstackSpec.Tenant, openstackSpec.Domain, datacenterName)
	}
}

func getOpenstackSizes(providers provider.CloudRegistry, username, passowrd, tenant, domain, datacenterName string) ([]apiv1.OpenstackSize, error) {
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
		apiSizes = append(apiSizes, apiSize)
	}

	return apiSizes, nil
}

func openstackTenantEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackTenantReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackTenantReq, got = %T", request)
		}

		return getOpenstackTenants(providers, req.Username, req.Password, req.Domain, req.DatacenterName)
	}
}

func openstackTenantNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
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

func openstackNetworkEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		return getOpenstackNetworks(providers, req.Username, req.Password, req.Tenant, req.Domain, req.DatacenterName)
	}
}

func openstackNetworkNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		openstackSpec := cluster.Spec.Cloud.Openstack
		datacenterName := cluster.Spec.Cloud.DatacenterName
		return getOpenstackNetworks(providers, openstackSpec.Username, openstackSpec.Password, openstackSpec.Tenant, openstackSpec.Domain, datacenterName)
	}
}

func getOpenstackNetworks(providers provider.CloudRegistry, username, password, tenant, domain, datacenterName string) ([]apiv1.OpenstackNetwork, error) {
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

func openstackSecurityGroupEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}
		return getOpenstackSecurityGroups(providers, req.Username, req.Password, req.Tenant, req.Domain, req.DatacenterName)
	}
}

func openstackSecurityGroupNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		openstackSpec := cluster.Spec.Cloud.Openstack
		datacenterName := cluster.Spec.Cloud.DatacenterName
		return getOpenstackSecurityGroups(providers, openstackSpec.Username, openstackSpec.Password, openstackSpec.Tenant, openstackSpec.Domain, datacenterName)
	}
}

func getOpenstackSecurityGroups(providers provider.CloudRegistry, username, password, tenant, domain, datacenterName string) ([]apiv1.OpenstackSecurityGroup, error) {
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

func openstackSubnetsEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackSubnetReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackSubnetReq, got = %T", request)
		}
		return getOpenstackSubnets(providers, req.Username, req.Password, req.Domain, req.Tenant, req.NetworkID, req.DatacenterName)
	}
}

func openstackSubnetsNoCredentialsEndpoint(projectProvider provider.ProjectProvider, providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(OpenstackSubnetNoCredentialsReq)
		cluster, err := getClusterForOpenstack(ctx, projectProvider, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		openstackSpec := cluster.Spec.Cloud.Openstack
		datacenterName := cluster.Spec.Cloud.DatacenterName
		return getOpenstackSubnets(providers, openstackSpec.Username, openstackSpec.Password, openstackSpec.Domain, openstackSpec.Tenant, req.NetworkID, datacenterName)
	}
}

func getOpenstackSubnets(providers provider.CloudRegistry, username, password, domain, tenant, networkID, datacenterName string) ([]apiv1.OpenstackSubnet, error) {
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
	clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
	userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
	_, err := projectProvider.Get(userInfo, projectID, &provider.ProjectGetOptions{})
	if err != nil {
		return nil, kubernetesErrorToHTTPError(err)
	}
	cluster, err := clusterProvider.Get(userInfo, clusterID, &provider.ClusterGetOptions{})
	if err != nil {
		return nil, kubernetesErrorToHTTPError(err)
	}
	if cluster.Spec.Cloud.Openstack == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}
	return cluster, nil
}
