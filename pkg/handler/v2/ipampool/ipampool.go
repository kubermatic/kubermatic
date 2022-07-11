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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// seedReq represents a request for referencing a seed
// swagger:parameters listIPAMPools
type seedReq struct {
	// in: path
	// required: true
	SeedName string `json:"seed_name"`
}

func (req seedReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.SeedName,
	}
}

func DecodeSeedReq(c context.Context, r *http.Request) (interface{}, error) {
	var req seedReq
	seedName := mux.Vars(r)["seed_name"]
	if seedName == "" {
		return nil, fmt.Errorf("'seed_name' parameter is required but was not provided")
	}
	req.SeedName = seedName
	return req, nil
}

// ipamPoolReq represents a request for managing a IPAM pool.
// swagger:parameters getIPAMPool deleteIPAMPool
type ipamPoolReq struct {
	seedReq

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
	req := ipamPoolReq{
		IPAMPoolName: mux.Vars(r)["ipampool_name"],
	}

	seedRequest, err := DecodeSeedReq(ctx, r)
	if err != nil {
		return nil, err
	}
	req.seedReq = seedRequest.(seedReq)

	return req, nil
}

// createIPAMPoolReq represents a request to create a IPAM pool
// swagger:parameters createIPAMPool
type createIPAMPoolReq struct {
	seedReq

	// in: body
	// required: true
	Body apiv2.IPAMPool
}

// Validate validates createIPAMPoolReq request.
func (r createIPAMPoolReq) Validate() error {
	if r.Body.Name == "" {
		return errors.New("missing attribute \"name\"")
	}
	if err := validateDatacenters(r.Body.Datacenters); err != nil {
		return err
	}
	return nil
}

func validateDatacenters(datacenters map[string]apiv2.IPAMPoolDatacenterSettings) error {
	if len(datacenters) == 0 {
		return errors.New("missing or empty attribute \"datacenters\"")
	}
	for dc, dcConfig := range datacenters {
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
	return nil
}

func DecodeCreateIPAMPoolReq(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createIPAMPoolReq

	seedRequest, err := DecodeSeedReq(ctx, r)
	if err != nil {
		return nil, err
	}
	req.seedReq = seedRequest.(seedReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// patchIPAMPoolReq represents a request to patch a IPAM pool
// swagger:parameters patchIPAMPool
type patchIPAMPoolReq struct {
	ipamPoolReq

	// in: body
	// required: true
	Body apiv2.IPAMPool
}

// Validate validates createIPAMPoolReq request.
func (r patchIPAMPoolReq) Validate() error {
	if err := r.ipamPoolReq.Validate(); err != nil {
		return err
	}
	if err := validateDatacenters(r.Body.Datacenters); err != nil {
		return err
	}
	return nil
}

func DecodePatchIPAMPoolReq(ctx context.Context, r *http.Request) (interface{}, error) {
	var req patchIPAMPoolReq

	ipamPoolRequest, err := DecodeIPAMPoolReq(ctx, r)
	if err != nil {
		return nil, err
	}
	req.ipamPoolReq = ipamPoolRequest.(ipamPoolReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func ListIPAMPoolsEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("%s doesn't have admin rights", userInfo.Email))
		}

		privilegedIPAMPoolProvider := ctx.Value(middleware.PrivilegedIPAMPoolProviderContextKey).(provider.PrivilegedIPAMPoolProvider)

		ipamPoolList, err := privilegedIPAMPoolProvider.ListUnsecured(ctx)
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

func GetIPAMPoolEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("%s doesn't have admin rights", userInfo.Email))
		}

		ipamPoolReq, ok := req.(ipamPoolReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := ipamPoolReq.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		privilegedIPAMPoolProvider := ctx.Value(middleware.PrivilegedIPAMPoolProviderContextKey).(provider.PrivilegedIPAMPoolProvider)

		ipamPool, err := privilegedIPAMPoolProvider.GetUnsecured(ctx, ipamPoolReq.IPAMPoolName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, utilerrors.NewNotFound("IPAMPool", ipamPoolReq.IPAMPoolName)
			}
			return nil, err
		}

		return toIPAMPoolAPIModel(ipamPool), nil
	}
}

func CreateIPAMPoolEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("%s doesn't have admin rights", userInfo.Email))
		}

		createIPAMPoolReq, ok := req.(createIPAMPoolReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := createIPAMPoolReq.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		ipamPool := toIPAMPoolKubermaticModel(&createIPAMPoolReq.Body)

		privilegedIPAMPoolProvider := ctx.Value(middleware.PrivilegedIPAMPoolProviderContextKey).(provider.PrivilegedIPAMPoolProvider)

		if err := privilegedIPAMPoolProvider.CreateUnsecured(ctx, ipamPool); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return nil, utilerrors.NewAlreadyExists("IPAMPool", createIPAMPoolReq.Body.Name)
			}
			return nil, err
		}

		return nil, nil
	}
}

func PatchIPAMPoolEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("%s doesn't have admin rights", userInfo.Email))
		}

		patchIPAMPoolReq, ok := req.(patchIPAMPoolReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := patchIPAMPoolReq.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		privilegedIPAMPoolProvider := ctx.Value(middleware.PrivilegedIPAMPoolProviderContextKey).(provider.PrivilegedIPAMPoolProvider)

		originalIPAMPool, err := privilegedIPAMPoolProvider.GetUnsecured(ctx, patchIPAMPoolReq.IPAMPoolName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, utilerrors.NewNotFound("IPAMPool", patchIPAMPoolReq.IPAMPoolName)
			}
			return nil, err
		}
		newIPAMPool := originalIPAMPool.DeepCopy()
		newIPAMPool.Spec = toIPAMPoolKubermaticModel(&patchIPAMPoolReq.Body).Spec

		if err := privilegedIPAMPoolProvider.PatchUnsecured(ctx, originalIPAMPool, newIPAMPool); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, utilerrors.NewNotFound("IPAMPool", patchIPAMPoolReq.IPAMPoolName)
			}
			return nil, err
		}

		return nil, nil
	}
}

func DeleteIPAMPoolEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("%s doesn't have admin rights", userInfo.Email))
		}

		ipamPoolReq, ok := req.(ipamPoolReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := ipamPoolReq.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		privilegedIPAMPoolProvider := ctx.Value(middleware.PrivilegedIPAMPoolProviderContextKey).(provider.PrivilegedIPAMPoolProvider)

		if err := privilegedIPAMPoolProvider.DeleteUnsecured(ctx, ipamPoolReq.IPAMPoolName); err != nil {
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
