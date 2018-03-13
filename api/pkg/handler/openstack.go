package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/openstack"
)

func openstackSizeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req, ok := request.(OpenstackSizeReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = OpenstackSizeReq, got = %T", request)
		}

		fakeDataCenterName := "nonExistingDataCenter"

		osProvider := openstack.NewCloudProvider(map[string]provider.DatacenterMeta{
			fakeDataCenterName: provider.DatacenterMeta{
				Spec: provider.DatacenterSpec{
					Openstack: &provider.OpenstackSpec{
						AuthURL: req.AuthURL,
						Region:  req.Region,
					},
				},
			},
		})

		flavors, err := osProvider.GetFlavors(&kubermaticv1.CloudSpec{
			DatacenterName: fakeDataCenterName,
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
