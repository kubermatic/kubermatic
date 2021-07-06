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

package whitelistedregistry

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

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, whitelistedRegistryProvider provider.PrivilegedWhitelistedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createWhitelistedRegistryReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		wr := &kubermaticv1.WhitelistedRegistry{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Body.Name,
			},
			Spec: req.Body.WhitelistedRegistrySpec,
		}

		wr, err = whitelistedRegistryProvider.CreateUnsecured(wr)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalWhitelistedRegistryToExternal(wr), nil
	}
}

// createWhitelistedRegistryReq represents a request for creating a whitelisted registry
// swagger:parameters createWhitelistedRegistry
type createWhitelistedRegistryReq struct {
	// in: body
	Body wrBody
}

type wrBody struct {
	// Name of the whitelisted registry
	Name string `json:"name"`
	// WhitelistedRegistrySpec Spec of the whitelisted registry
	WhitelistedRegistrySpec kubermaticv1.WhitelistedRegistrySpec `json:"spec"`
}

func DecodeCreateWhitelistedRegistryRequest(c context.Context, r *http.Request) (interface{}, error) {
	var req createWhitelistedRegistryReq

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func GetEndpoint(userInfoGetter provider.UserInfoGetter, whitelistedRegistryProvider provider.PrivilegedWhitelistedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getWhitelistedRegistryReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		wr, err := whitelistedRegistryProvider.GetUnsecured(req.WhitelistedRegistryName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalWhitelistedRegistryToExternal(wr), nil
	}
}

// getWhitelistedRegistryReq represents a request for getting a whitelisted registry
// swagger:parameters getWhitelistedRegistry deleteWhitelistedRegistry
type getWhitelistedRegistryReq struct {
	// in: path
	// required: true
	WhitelistedRegistryName string `json:"whitelisted_registry"`
}

func DecodeGetWhitelistedRegistryRequest(c context.Context, r *http.Request) (interface{}, error) {
	var req getWhitelistedRegistryReq

	whitelistedRegistry := mux.Vars(r)["whitelisted_registry"]
	if whitelistedRegistry == "" {
		return "", fmt.Errorf("'whitelisted_registry' parameter is required but was not provided")
	}
	req.WhitelistedRegistryName = whitelistedRegistry

	return req, nil
}

func ListEndpoint(userInfoGetter provider.UserInfoGetter, whitelistedRegistryProvider provider.PrivilegedWhitelistedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		whitelistedRegistryList, err := whitelistedRegistryProvider.ListUnsecured()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiWR := make([]*apiv2.WhitelistedRegistry, 0)
		for _, wr := range whitelistedRegistryList.Items {
			apiWR = append(apiWR, convertInternalWhitelistedRegistryToExternal(&wr))
		}

		return apiWR, nil
	}
}

func convertInternalWhitelistedRegistryToExternal(wr *kubermaticv1.WhitelistedRegistry) *apiv2.WhitelistedRegistry {
	return &apiv2.WhitelistedRegistry{
		Name: wr.Name,
		Spec: wr.Spec,
	}
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter, whitelistedRegistryProvider provider.PrivilegedWhitelistedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getWhitelistedRegistryReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		err = whitelistedRegistryProvider.DeleteUnsecured(req.WhitelistedRegistryName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

// patchWhitelistedRegistryReq defines HTTP request for patching whitelisted registries
// swagger:parameters patchWhitelistedRegistry
type patchWhitelistedRegistryReq struct {
	getWhitelistedRegistryReq
	// in: body
	Patch json.RawMessage
}

// DecodePatchWhitelistedRegistryReq decodes http request into patchWhitelistedRegistryReq
func DecodePatchWhitelistedRegistryReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchWhitelistedRegistryReq

	wrReq, err := DecodeGetWhitelistedRegistryRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.getWhitelistedRegistryReq = wrReq.(getWhitelistedRegistryReq)

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func PatchEndpoint(userInfoGetter provider.UserInfoGetter, whitelistedRegistryProvider provider.PrivilegedWhitelistedRegistryProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchWhitelistedRegistryReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		// get WR
		originalWR, err := whitelistedRegistryProvider.GetUnsecured(req.WhitelistedRegistryName)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		originalAPIWR := convertInternalWhitelistedRegistryToExternal(originalWR)

		// patch
		originalJSON, err := json.Marshal(originalAPIWR)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to convert current whitelistdRegistry: %v", err))
		}

		patchedJSON, err := jsonpatch.MergePatch(originalJSON, req.Patch)
		if err != nil {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("failed to merge patch whitelistdRegistry: %v", err))
		}

		var patched *apiv2.WhitelistedRegistry
		err = json.Unmarshal(patchedJSON, &patched)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal patch whitelistdRegistry: %v", err))
		}

		// validate
		if patched.Name != originalWR.Name {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("Changing whitelistedRegistry name is not allowed: %q to %q", originalWR.Name, patched.Name))
		}

		patchedCT := &kubermaticv1.WhitelistedRegistry{
			ObjectMeta: metav1.ObjectMeta{
				Name:            patched.Name,
				ResourceVersion: originalWR.ResourceVersion,
			},
			Spec: patched.Spec,
		}

		// apply patch
		patchedCT, err = whitelistedRegistryProvider.PatchUnsecured(patchedCT)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalWhitelistedRegistryToExternal(patchedCT), nil
	}
}
