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
	"net/http"

	"github.com/go-kit/kit/endpoint"

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

func convertInternalWhitelistedRegistryToExternal(wr *kubermaticv1.WhitelistedRegistry) *apiv2.WhitelistedRegistry {
	return &apiv2.WhitelistedRegistry{
		Name: wr.Name,
		Spec: wr.Spec,
	}
}
