package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
)

func openstackSizeEndpoint(providers provider.CloudRegistry, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
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

		dc, found := dcs[req.DatacenterName]
		if !found {
			return nil, fmt.Errorf("invalid node datacenter '%s'", req.DatacenterName)
		}

		flavors, err := osProvider.GetFlavors(&kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				AuthURL:  dc.Spec.Openstack.AuthURL,
				Region:   dc.Spec.Openstack.Region,
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

func openstackTenantEndpoint(providers provider.CloudRegistry, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
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

		dc, found := dcs[req.DatacenterName]
		if !found {
			return nil, fmt.Errorf("invalid node datacenter '%s'", req.DatacenterName)
		}

		tenants, err := osProvider.GetTenants(&kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				AuthURL:  dc.Spec.Openstack.AuthURL,
				Region:   dc.Spec.Openstack.Region,
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
