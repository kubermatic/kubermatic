package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
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

		osProvider, ok := osProviderInterface.(*openstack.Openstack)
		if !ok {
			return nil, fmt.Errorf("unable to cast osProviderInterface to *openstack.Openstack")
		}

		flavors, err := osProvider.GetFlavors(&kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Username: req.Username,
				Password: req.Password,
				Tenant:   req.Tenant,
				Domain:   req.Domain,
			},
		}, req.Region)
		if err != nil {
			return nil, err
		}

		return flavors, nil
	}
}
