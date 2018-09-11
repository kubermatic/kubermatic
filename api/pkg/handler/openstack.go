package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
)

func openstackSizeEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
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

		flavors, dc, err := osProvider.GetFlavors(kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Username: req.Username,
				Password: req.Password,
				Tenant:   req.Tenant,
				Domain:   req.Domain,
			},
		})
		if err != nil {
			return nil, err
		}

		apiSizes := []apiv1.OpenstackSize{}
		for _, flavor := range flavors {
			apiSize := apiv1.OpenstackSize{
				ID:       flavor.ID,
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
		req, ok := request.(OpenstackTenantReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackTenantReq, got = %T", request)
		}

		osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
		if !ok {
			return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
		}

		osProvider, ok := osProviderInterface.(*openstack.Provider)
		if !ok {
			return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
		}

		tenants, err := osProvider.GetTenants(kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Username: req.Username,
				Password: req.Password,
				Domain:   req.Domain,
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

		networks, err := osProvider.GetNetworks(kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Username: req.Username,
				Password: req.Password,
				Tenant:   req.Tenant,
				Domain:   req.Domain,
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
}

func openstackSecurityGroupEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
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

		securityGroups, err := osProvider.GetSecurityGroups(kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Username: req.Username,
				Password: req.Password,
				Tenant:   req.Tenant,
				Domain:   req.Domain,
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
}

func openstackSubnetsEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackSubnetReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackSubnetReq, got = %T", request)
		}

		osProviderInterface, ok := providers[provider.OpenstackCloudProvider]
		if !ok {
			return nil, fmt.Errorf("unable to get %s provider", provider.OpenstackCloudProvider)
		}

		osProvider, ok := osProviderInterface.(*openstack.Provider)
		if !ok {
			return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Provider")
		}

		subnets, err := osProvider.GetSubnets(kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Username: req.Username,
				Password: req.Password,
				Domain:   req.Domain,
				Tenant:   req.Tenant,
			},
		}, req.NetworkID)
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
