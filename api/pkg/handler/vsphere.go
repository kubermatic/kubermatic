package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/vsphere"
)

func vsphereNetworksEndpoint(providers provider.CloudRegistry) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req, ok := request.(VSphereNetworksReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = VSphereNetworksReq, got = %T", request)
		}

		vsProviderInterface, ok := providers[provider.VSphereCloudProvider]
		if !ok {
			return nil, fmt.Errorf("unable to get %s provider", provider.VSphereCloudProvider)
		}

		vsProvider, ok := vsProviderInterface.(*vsphere.Provider)
		if !ok {
			return nil, fmt.Errorf("unable to cast vsProviderInterface to *vsphere.Provider")
		}

		networks, err := vsProvider.GetNetworks(&kubermaticv1.CloudSpec{
			DatacenterName: req.DatacenterName,
			VSphere: &kubermaticv1.VSphereCloudSpec{
				Username: req.Username,
				Password: req.Password,
			},
		})
		if err != nil {
			return nil, err
		}

		return networks, nil
	}
}
