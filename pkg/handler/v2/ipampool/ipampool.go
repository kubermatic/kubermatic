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
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ipamPoolReq represents a request for managing a IPAM pool.
// swagger:parameters getIPAMPool deleteIPAMPool
type ipamPoolReq struct {
	// in: path
	// required: true
	IPAMPoolName string `json:"ipampool_name"`
}

// Validate validates ipamPoolReq request.
func (r ipamPoolReq) Validate() error {
	if r.IPAMPoolName == "" {
		return fmt.Errorf("the IPAM pool name cannot be empty")
	}
	return nil
}

func DecodeIPAMPoolReq(ctx context.Context, r *http.Request) (interface{}, error) {
	return ipamPoolReq{
		IPAMPoolName: mux.Vars(r)["ipampool_name"],
	}, nil
}

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
			resp[i] = toIPAMPoolAPIModel(&ipamPool)
		}

		return resp, nil
	}
}

func GetIPAMPoolEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.IPAMPoolProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		ipamPoolReq, ok := req.(ipamPoolReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := ipamPoolReq.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		ipamPool, err := provider.Get(ctx, ipamPoolReq.IPAMPoolName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, utilerrors.NewNotFound("IPAMPool", ipamPoolReq.IPAMPoolName)
			}
			return nil, err
		}

		return toIPAMPoolAPIModel(ipamPool), nil
	}
}

func DeleteIPAMPoolEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.IPAMPoolProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		ipamPoolReq, ok := req.(ipamPoolReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := ipamPoolReq.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		if err := provider.Delete(ctx, ipamPoolReq.IPAMPoolName); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, utilerrors.NewNotFound("IPAMPool", ipamPoolReq.IPAMPoolName)
			}
			return nil, err
		}

		return nil, nil
	}
}

func toIPAMPoolAPIModel(ipamPool *kubermaticv1.IPAMPool) *apiv2.IPAMPool {
	apiIPAMPool := &apiv2.IPAMPool{
		Name:        ipamPool.Name,
		Datacenters: make(map[string]apiv2.IPAMPoolDatacenterSettings, len(ipamPool.Spec.Datacenters)),
	}

	for dc, dcConfig := range ipamPool.Spec.Datacenters {
		apiIPAMPool.Datacenters[dc] = apiv2.IPAMPoolDatacenterSettings{
			Type:             dcConfig.Type,
			PoolCIDR:         dcConfig.PoolCIDR,
			AllocationPrefix: dcConfig.AllocationPrefix,
			AllocationRange:  dcConfig.AllocationRange,
		}
	}

	return apiIPAMPool
}
