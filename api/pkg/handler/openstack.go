package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
)

func openstackSizeEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req, ok := request.(OpenstackSizeReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackSizeReq, got = %T", request)
		}

		osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
		if !ok {
			return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
		}

		osProvider, ok := osProviderInterface.(*openstack.Provider)
		if !ok {
			return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
		}

		serviceClient, err := osProvider.ServiceClient(req.Username, req.Password, req.Domain, req.DatacenterName)
		if err != nil {
			return nil, err
		}

		flavors, dc, err := osProvider.GetFlavors(serviceClient, req.DatacenterName)
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
}

func openstackTenantEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}

		osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
		if !ok {
			return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
		}

		osProvider, ok := osProviderInterface.(*openstack.Provider)
		if !ok {
			return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
		}

		serviceClient, err := osProvider.ServiceClient(req.Username, req.Password, req.Domain, req.DatacenterName)
		if err != nil {
			return nil, err
		}

		tenants, err := osProvider.GetTenants(serviceClient, req.DatacenterName)
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
}

func openstackNetworkEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}

		osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
		if !ok {
			return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
		}

		osProvider, ok := osProviderInterface.(*openstack.Provider)
		if !ok {
			return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
		}

		serviceClient, err := osProvider.ServiceClient(req.Username, req.Password, req.Domain, req.DatacenterName)
		if err != nil {
			return nil, err
		}

		networks, err := osProvider.GetNetworks(serviceClient)
		if err != nil {
			return nil, err
		}

		apiNetworks := []apiv1.OpenstackNetwork{}
		for _, network := range networks {
			apiNetwork := apiv1.OpenstackNetwork{
				Name: network.Name,
				ID:   network.ID,
			}

			apiNetworks = append(apiNetworks, apiNetwork)
		}

		return apiNetworks, nil
	}
}

func openstackSubnetsEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackReq, got = %T", request)
		}

		osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
		if !ok {
			return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
		}

		osProvider, ok := osProviderInterface.(*openstack.Provider)

		if !ok {
			return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
		}

		serviceClient, err := osProvider.ServiceClient(req.Username, req.Password, req.Domain, req.DatacenterName)
		if err != nil {
			return nil, err
		}

		subnets, err := osProvider.GetSubnetIDs(serviceClient)
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
}
