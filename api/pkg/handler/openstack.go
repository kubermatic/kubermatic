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

		flavors, err := osProvider.GetFlavors(&kubermaticv1.CloudSpec{
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

		return flavors, nil
	}
}

func openstackTenantEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(OpenstackReq)
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

		tenants, err := osProvider.GetTenants(&kubermaticv1.CloudSpec{
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

		return tenants, nil
	}
}

// /api/v1/openstack/subnets
func openstackSubnetIDsEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
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

		subnetIDs, err := osProvider.GetSubnetIDs(&kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Username: req.Username,
				Password: req.Password,
				Domain:   req.Domain,
			},
		})
		if err != nil {
			return nil, err
		}

		apiSubnetIDs := []apiv1.OpenstackSubnetID{}
		for _, sunbetID := range subnetIDs {
			apiSubnetIDs = append(apiSubnetIDs, apiv1.OpenstackSubnetID{
				ID:   sunbetID.ID,
				Name: sunbetID.Name,
			})
		}

		return apiSubnetIDs, nil
	}
}
