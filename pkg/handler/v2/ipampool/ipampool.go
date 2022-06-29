/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ipampool

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func ListIPAMPoolsEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.IPAMPoolProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		ipamPoolList, err := provider.List(ctx)
		if err != nil {
			return nil, err
		}

		resp := make([]*apiv2.IPAMPool, len(ipamPoolList.Items))
		for i, ipamPool := range ipamPoolList.Items {
			resp[i] = &apiv2.IPAMPool{
				Name:        ipamPool.Name,
				Datacenters: make(map[string]apiv2.IPAMPoolDatacenterSettings, len(ipamPool.Spec.Datacenters)),
			}

			for dc, dcConfig := range ipamPool.Spec.Datacenters {
				resp[i].Datacenters[dc] = apiv2.IPAMPoolDatacenterSettings{
					Type:             dcConfig.Type,
					PoolCIDR:         dcConfig.PoolCIDR,
					AllocationPrefix: dcConfig.AllocationPrefix,
					AllocationRange:  dcConfig.AllocationRange,
				}
			}
		}

		return resp, nil
	}
}
