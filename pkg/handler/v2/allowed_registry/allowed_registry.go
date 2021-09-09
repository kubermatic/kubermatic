/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package allowedregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, allowedRegistryProvider provider.PrivilegedAllowedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createAllowedRegistryReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		wr := &kubermaticv1.AllowedRegistry{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Body.Name,
			},
			Spec: req.Body.AllowedRegistrySpec,
		}

		wr, err = allowedRegistryProvider.CreateUnsecured(wr)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalAllowedRegistryToExternal(wr), nil
	}
}

// createAllowedRegistryReq represents a request for creating a allowed registry
// swagger:parameters createAllowedRegistry
type createAllowedRegistryReq struct {
	// in: body
	Body wrBody
}

type wrBody struct {
	// Name of the allowed registry
	Name string `json:"name"`
	// AllowedRegistrySpec Spec of the allowed registry
	AllowedRegistrySpec kubermaticv1.AllowedRegistrySpec `json:"spec"`
}

func DecodeCreateAllowedRegistryRequest(c context.Context, r *http.Request) (interface{}, error) {
	var req createAllowedRegistryReq

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func GetEndpoint(userInfoGetter provider.UserInfoGetter, allowedRegistryProvider provider.PrivilegedAllowedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getAllowedRegistryReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		wr, err := allowedRegistryProvider.GetUnsecured(req.AllowedRegistryName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalAllowedRegistryToExternal(wr), nil
	}
}

// getAllowedRegistryReq represents a request for getting a allowed registry
// swagger:parameters getAllowedRegistry deleteAllowedRegistry
type getAllowedRegistryReq struct {
	// in: path
	// required: true
	AllowedRegistryName string `json:"allowed_registry"`
}

func DecodeGetAllowedRegistryRequest(c context.Context, r *http.Request) (interface{}, error) {
	var req getAllowedRegistryReq

	allowedRegistry := mux.Vars(r)["allowed_registry"]
	if allowedRegistry == "" {
		return "", fmt.Errorf("'allowed_registry' parameter is required but was not provided")
	}
	req.AllowedRegistryName = allowedRegistry

	return req, nil
}

func ListEndpoint(userInfoGetter provider.UserInfoGetter, allowedRegistryProvider provider.PrivilegedAllowedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		allowedRegistryList, err := allowedRegistryProvider.ListUnsecured()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiWR := make([]*apiv2.AllowedRegistry, 0)
		for _, wr := range allowedRegistryList.Items {
			apiWR = append(apiWR, convertInternalAllowedRegistryToExternal(&wr))
		}

		return apiWR, nil
	}
}

func convertInternalAllowedRegistryToExternal(wr *kubermaticv1.AllowedRegistry) *apiv2.AllowedRegistry {
	return &apiv2.AllowedRegistry{
		Name: wr.Name,
		Spec: wr.Spec,
	}
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter, allowedRegistryProvider provider.PrivilegedAllowedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getAllowedRegistryReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		err = allowedRegistryProvider.DeleteUnsecured(req.AllowedRegistryName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

// patchAllowedRegistryReq defines HTTP request for patching allowed registries
// swagger:parameters patchAllowedRegistry
type patchAllowedRegistryReq struct {
	getAllowedRegistryReq
	// in: body
	Patch json.RawMessage
}

// DecodePatchAllowedRegistryReq decodes http request into patchAllowedRegistryReq
func DecodePatchAllowedRegistryReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchAllowedRegistryReq

	wrReq, err := DecodeGetAllowedRegistryRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.getAllowedRegistryReq = wrReq.(getAllowedRegistryReq)

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func PatchEndpoint(userInfoGetter provider.UserInfoGetter, allowedRegistryProvider provider.PrivilegedAllowedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchAllowedRegistryReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		// get WR
		allowedRegistry, err := allowedRegistryProvider.GetUnsecured(req.AllowedRegistryName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		originalAPIAR := convertInternalAllowedRegistryToExternal(allowedRegistry)

		// patch
		originalJSON, err := json.Marshal(originalAPIAR)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to convert current allowedRegistry: %v", err))
		}

		patchedJSON, err := jsonpatch.MergePatch(originalJSON, req.Patch)
		if err != nil {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("failed to merge patch alloweddRegistry: %v", err))
		}

		var patched *apiv2.AllowedRegistry
		err = json.Unmarshal(patchedJSON, &patched)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal patch allowedRegistry: %v", err))
		}

		// validate
		if patched.Name != allowedRegistry.Name {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("Changing allowedRegistry name is not allowed: %q to %q", allowedRegistry.Name, patched.Name))
		}

		allowedRegistry.Spec = patched.Spec

		// apply patch
		allowedRegistry, err = allowedRegistryProvider.UpdateUnsecured(allowedRegistry)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalAllowedRegistryToExternal(allowedRegistry), nil
	}
}
