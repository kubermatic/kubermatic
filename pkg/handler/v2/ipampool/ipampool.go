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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		return errors.New("the IPAM pool name cannot be empty")
	}
	return nil
}

func DecodeIPAMPoolReq(ctx context.Context, r *http.Request) (interface{}, error) {
	return ipamPoolReq{
		IPAMPoolName: mux.Vars(r)["ipampool_name"],
	}, nil
}

// createUpdateIPAMPoolReq represents a request to create or update a IPAM pool
// swagger:parameters createIPAMPool updateIPAMPool
type createUpdateIPAMPoolReq struct {
	// in: body
	// required: true
	Body apiv2.IPAMPool
}

// Validate validates createUpdateIPAMPoolReq request.
func (r createUpdateIPAMPoolReq) Validate() error {
	if r.Body.Name == "" {
		return errors.New("missing attribute \"name\"")
	}
	if len(r.Body.Datacenters) == 0 {
		return errors.New("missing or empty attribute \"datacenters\"")
	}
	for dc, dcConfig := range r.Body.Datacenters {
		if dcConfig.PoolCIDR == "" {
			return fmt.Errorf("missing attribute \"poolCidr\" for datacenter %s", dc)
		}
		if dcConfig.Type == "" {
			return fmt.Errorf("missing attribute \"type\" for datacenter %s", dc)
		}
		switch dcConfig.Type {
		case kubermaticv1.IPAMPoolAllocationTypeRange:
			if dcConfig.AllocationRange == 0 {
				return fmt.Errorf("missing attribute \"allocationRange\" for datacenter %s", dc)
			}
		case kubermaticv1.IPAMPoolAllocationTypePrefix:
			if dcConfig.AllocationPrefix == 0 {
				return fmt.Errorf("missing attribute \"allocationPrefix\" for datacenter %s", dc)
			}
		}
	}
	// TODO: same webhook validations here
	return nil
}

func DecodeCreateUpdateIPAMPoolReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req createUpdateIPAMPoolReq

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
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

func CreateIPAMPoolEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.IPAMPoolProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		createIPAMPoolReq, ok := req.(createUpdateIPAMPoolReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := createIPAMPoolReq.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		if err := provider.Create(ctx, toIPAMPoolKubermaticModel(&createIPAMPoolReq.Body)); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return nil, utilerrors.NewAlreadyExists("IPAMPool", createIPAMPoolReq.Body.Name)
			}
			return nil, err
		}

		return nil, nil
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

func toIPAMPoolKubermaticModel(ipamPool *apiv2.IPAMPool) *kubermaticv1.IPAMPool {
	kubermaticIPAMPool := &kubermaticv1.IPAMPool{
		ObjectMeta: metav1.ObjectMeta{
			Name: ipamPool.Name,
		},
		Spec: kubermaticv1.IPAMPoolSpec{
			Datacenters: make(map[string]kubermaticv1.IPAMPoolDatacenterSettings, len(ipamPool.Datacenters)),
		},
	}

	for dc, dcConfig := range ipamPool.Datacenters {
		kubermaticIPAMPool.Spec.Datacenters[dc] = kubermaticv1.IPAMPoolDatacenterSettings{
			Type:             dcConfig.Type,
			PoolCIDR:         dcConfig.PoolCIDR,
			AllocationPrefix: dcConfig.AllocationPrefix,
			AllocationRange:  dcConfig.AllocationRange,
		}
	}

	return kubermaticIPAMPool
}
