/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package admin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// GetAdminEndpoint returns list of admin users.
func GetAdminEndpoint(userInfoGetter provider.UserInfoGetter, adminProvider provider.AdminProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		admins, err := adminProvider.GetAdmins(ctx, userInfo)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var resultList []apiv1.Admin
		for _, admin := range admins {
			resultList = append(resultList, apiv1.Admin{Email: admin.Spec.Email, IsAdmin: admin.Spec.IsAdmin, Name: admin.Spec.Name})
		}

		return resultList, nil
	}
}

// SetAdminEndpoint allows setting and clearing admin role for users.
func SetAdminEndpoint(userInfoGetter provider.UserInfoGetter, adminProvider provider.AdminProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(setAdminReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, err
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		admin, err := adminProvider.SetAdmin(ctx, userInfo, req.Body.Email, req.Body.IsAdmin)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return apiv1.Admin{
			Email:   admin.Spec.Email,
			Name:    admin.Spec.Name,
			IsAdmin: admin.Spec.IsAdmin,
		}, nil
	}
}

// setAdminReq defines HTTP request for setAdmin
// swagger:parameters setAdmin
type setAdminReq struct {
	// in: body
	Body apiv1.Admin
}

// Validate setAdminReq request.
func (r setAdminReq) Validate() error {
	if len(r.Body.Email) == 0 {
		return utilerrors.NewBadRequest("the email address cannot be empty")
	}
	return nil
}

// DecodeSetAdminReq  decodes an HTTP request into setAdminReq.
func DecodeSetAdminReq(c context.Context, r *http.Request) (interface{}, error) {
	var req setAdminReq
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}
